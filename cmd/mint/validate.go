package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/color"
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
		cp := color.New()
		for _, d := range result.Diagnostics {
			label := cp.SeverityLabel(d.Severity)
			ruleID := ""
			if d.RuleID != "" {
				ruleID = " " + cp.Gray(d.RuleID)
			}
			if d.Path != "" {
				fmt.Printf("%s %s: %s%s\n", label, d.Path, d.Message, ruleID)
			} else {
				fmt.Printf("%s %s%s\n", label, d.Message, ruleID)
			}
		}
		if result.Valid {
			fmt.Println(cp.Bold("Spec is valid."))
		} else {
			fmt.Println(cp.Error("Spec has errors."))
		}
	}

	if !result.Valid {
		return 1
	}
	return 0
}
