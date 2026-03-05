package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/merge"
)

func runMerge(args []string) int {
	fs := flag.NewFlagSet("mint merge", flag.ContinueOnError)
	output := fs.String("o", "", "Output file (default: stdout)")
	onConflict := fs.String("on-conflict", "fail", "Conflict strategy: fail, skip, or rename")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "error: at least 2 spec files required\n\nUsage: mint merge [flags] <spec1> <spec2> [spec3...]")
		return 1
	}

	var specs [][]byte
	for _, path := range fs.Args() {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
			return 1
		}
		specs = append(specs, data)
	}

	strategy := merge.ConflictStrategy(*onConflict)
	result, err := merge.Specs(specs, strategy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		for _, c := range result.Conflicts {
			fmt.Fprintf(os.Stderr, "  %s\n", c)
		}
		return 1
	}

	if *output != "" {
		if err := os.WriteFile(*output, result.Output, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "Merged spec written to %s\n", *output)
	} else {
		fmt.Print(string(result.Output))
	}

	if len(result.Conflicts) > 0 {
		fmt.Fprintf(os.Stderr, "%d conflict(s) resolved.\n", len(result.Conflicts))
	}

	return 0
}
