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
	pivot := flag.String("pivot", "", "print a pivot table as CSV, e.g. rows=region,cols=status,values=amount:sum,*:count,totals=true")
	out := flag.StringP("output", "o", "", "with --pivot, write to a .csv, .parquet or .ndjson file instead of stdout")
	source := flag.String("source-column", "", "add a column with each row's file name, named `name` (default \"source\")")
	flag.Lookup("source-column").NoOptDefVal = "source"
	flag.Parse()
	if *showVersion {
		fmt.Println("pivotlens", version)
		return
	}
	paths := flag.Args()
	if len(paths) == 0 && !isTerminal(os.Stdin) {
		paths = []string{engine.StdinPath}
	}
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "usage: pivotlens [--pivot spec [-o out.csv]] [--source-column[=name]] <file|glob|->...")
		os.Exit(2)
	}
	in := engine.Input{Paths: paths, Stdin: os.Stdin, Source: *source}
	var err error
	switch {
	case *pivot != "":
		err = runPivot(in, *pivot, *out)
	case *out != "":
		err = errors.New("--output needs --pivot")
	default:
		err = run(in)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "pivotlens:", err)
		os.Exit(1)
	}
}

func run(in engine.Input) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
		}
	}()
	sess, err := engine.OpenInput(context.Background(), in)
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close() }()
	_, err = tea.NewProgram(ui.New(sess)).Run()
	return err
}

func runPivot(in engine.Input, spec, out string) error {
	p, err := engine.ParsePivot(spec)
	if err != nil {
		return err
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(out)), ".")
	if out != "" && format != "csv" && format != "parquet" && format != "ndjson" {
		return errors.New("--output must end in .csv, .parquet or .ndjson")
	}
	ctx := context.Background()
	sess, err := engine.OpenInput(ctx, in)
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

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err != nil || fi.Mode()&os.ModeCharDevice != 0
}
