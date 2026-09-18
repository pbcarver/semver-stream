// Command semvercheck validates every line of its input (or of the files
// named on the command line) as a semantic version and prints the
// canonical form of each one that parses.
package main

import (
	"fmt"
	"io"
	"os"

	semver "github.com/pbcarver/semver-stream"
)

type namedReader struct {
	name string
	r    io.Reader
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var sources []namedReader
	if len(args) == 0 {
		sources = []namedReader{{"stdin", os.Stdin}}
	} else {
		for _, path := range args {
			f, err := os.Open(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			defer f.Close()
			sources = append(sources, namedReader{path, f})
		}
	}

	exit := 0
	for _, src := range sources {
		sc := semver.NewScanner(src.r)
		for sc.Next() {
			res := sc.Result()
			if res.Err != nil {
				fmt.Fprintf(os.Stderr, "%s:%d: %v\n", src.name, res.Line, res.Err)
				exit = 1
				continue
			}
			fmt.Printf("%s:%d: %s\n", src.name, res.Line, res.Version.String())
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", src.name, err)
			exit = 1
		}
	}
	return exit
}
