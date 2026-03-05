package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/diff"
	"github.com/sirerun/mint/internal/loader"
)

func runDiff(args []string) int {
	fs := flag.NewFlagSet("mint diff", flag.ContinueOnError)
	format := fs.String("format", "text", "Output format: text or json")
	failOnBreaking := fs.Bool("fail-on-breaking", false, "Exit with code 1 if breaking changes found")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "error: two spec files required\n\nUsage: mint diff [flags] <old-spec> <new-spec>")
		return 1
	}

	oldResult, err := loader.Load(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading old spec: %v\n", err)
		return 1
	}

	newResult, err := loader.Load(fs.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading new spec: %v\n", err)
		return 1
	}

	result := diff.Specs(oldResult.Model, newResult.Model)

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			return 1
		}
	} else {
		if len(result.Changes) == 0 {
			fmt.Println("No changes found.")
		} else {
			for _, c := range result.Changes {
				fmt.Println(c.String())
			}
			fmt.Printf("\n%d change(s), %d breaking.\n", result.TotalChanges, result.BreakingChanges)
		}
	}

	if *failOnBreaking && result.HasBreaking {
		return 1
	}
	return 0
}
