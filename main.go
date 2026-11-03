package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	flag "github.com/spf13/pflag"

	"github.com/sv222/pivotlens/internal/engine"
	"github.com/sv222/pivotlens/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.BoolP("version", "v", false, "print version and exit")
	pivot := flag.String("pivot", "", "print a pivot table as CSV, e.g. rows=region,cols=status,values=amount,agg=sum")
	out := flag.StringP("output", "o", "", "with --pivot, write to a .csv or .parquet file instead of stdout")
	flag.Parse()
	if *showVersion {
		fmt.Println("pivotlens", version)
		return
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: pivotlens [--pivot spec [-o out.csv]] <file.csv|file.parquet|file.ndjson>")
		os.Exit(2)
	}
	var err error
	switch {
	case *pivot != "":
		err = runPivot(flag.Arg(0), *pivot, *out)
	case *out != "":
		err = errors.New("--output needs --pivot")
	default:
		err = run(flag.Arg(0))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "pivotlens:", err)
		os.Exit(1)
	}
}

func run(path string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	sess, err := engine.Open(context.Background(), path)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()
	_, err = tea.NewProgram(ui.New(sess)).Run()
	return err
}

func runPivot(path, spec, out string) error {
	p, err := engine.ParsePivot(spec)
	if err != nil {
		return err
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(out)), ".")
	if out != "" && format != "csv" && format != "parquet" {
		return errors.New("--output must end in .csv or .parquet")
	}
	ctx := context.Background()
	sess, err := engine.Open(ctx, path)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()
	q := engine.NewSpec(sess.Columns())
	q.Pivot = &p
	if q, err = sess.ResolvePivot(ctx, q); err != nil {
		return err
	}
	if out == "" {
		return sess.WriteCSV(ctx, q, os.Stdout)
	}
	return sess.Export(ctx, q, out, format)
}
