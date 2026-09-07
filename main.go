package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	flag "github.com/spf13/pflag"

	"github.com/sv222/pivotlens/internal/engine"
	"github.com/sv222/pivotlens/internal/ui"
)

var version = "dev"

func main() {
	showVersion := flag.BoolP("version", "v", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("pivotlens", version)
		return
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: pivotlens <file.csv|file.parquet|file.ndjson>")
		os.Exit(2)
	}
	if err := run(flag.Arg(0)); err != nil {
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
	defer sess.Close()
	_, err = tea.NewProgram(ui.New(sess)).Run()
	return err
}
