// The fumpt program formats the YAML source in a Fleet integration. It
// canonicalises YAML map field order and quote usage where possible.
// In field definition files it orders field definitions lexically by
// mapping name.
//
// Without an explicit path, it processes the package containing the
// working directory, otherwise it processes the package containing
// the directory of the command line argument.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	write := flag.Bool("w", false, "write result to (source) files instead of stdout")
	help := flag.Bool("h", false, "display help")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}
	flag.Parse()
	if *help {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage of %s:

The fumpt program formats the YAML source in a Fleet integration. It
canonicalises YAML map field order and quote usage where possible.
In field definition files it orders field definitions lexically by
mapping name.

Without an explicit path, it processes the package containing the
working directory, otherwise it processes the package containing
the directory of the command line argument.

By default the formatting is written to standard output as a txtar
archive for inspection.

`, os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	paths := []string{"."}
	if len(os.Args) > 1 {
		paths = os.Args[1:]
	}
	for _, p := range paths {
		r, err := root(p)
		if err != nil {
			switch {
			case errors.Is(err, os.ErrNotExist):
				if p == "." {
					fmt.Fprintln(os.Stderr, "not in a package")
				} else {
					fmt.Fprintf(os.Stderr, "%s is not in a package\n", filepath.Clean(p))
				}
			default:
				fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
			}
			continue
		}

		var w io.Writer
		if !*write {
			w = os.Stdout
		}
		err = walk(r, w, conventions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		}
	}
}
