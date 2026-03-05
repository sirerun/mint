package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/validate"
)

func runValidate(args []string) int {
	fs := flag.NewFlagSet("mint validate", flag.ContinueOnError)
	format := fs.String("format", "text", "Output format: text or json")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: spec file path required\n\nUsage: mint validate [flags] <spec-file>")
		return 1
	}

	specPath := fs.Arg(0)
	data, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		return 1
	}

	result := validate.Spec(data, specPath)

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			return 1
		}
	} else {
		for _, d := range result.Diagnostics {
			fmt.Println(d.String())
		}
		if result.Valid {
			fmt.Println("Spec is valid.")
		} else {
			fmt.Println("Spec has errors.")
		}
	}

	if !result.Valid {
		return 1
	}
	return 0
}
