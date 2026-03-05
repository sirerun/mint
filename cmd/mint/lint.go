package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirerun/mint/internal/color"
	"github.com/sirerun/mint/internal/lint"
)

func runLint(args []string) int {
	fs := flag.NewFlagSet("mint lint", flag.ContinueOnError)
	format := fs.String("format", "text", "Output format: text or json")
	ruleset := fs.String("ruleset", "recommended", "Ruleset: minimal, recommended, or strict")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: spec file path required\n\nUsage: mint lint [flags] <spec-file>")
		return 1
	}

	rs, ok := lint.GetRuleset(*ruleset)
	if !ok {
		fmt.Fprintf(os.Stderr, "error: unknown ruleset %q (available: minimal, recommended, strict)\n", *ruleset)
		return 1
	}

	specPath := fs.Arg(0)
	data, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		return 1
	}

	result, err := lint.Run(data, specPath, rs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if *format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			return 1
		}
	} else {
		cp := color.New()
		for _, d := range result.Items {
			label := cp.SeverityLabel(d.Severity)
			ruleID := cp.Gray(d.RuleID)
			if d.Path != "" {
				fmt.Printf("%s %s: %s %s\n", label, d.Path, d.Message, ruleID)
			} else {
				fmt.Printf("%s %s %s\n", label, d.Message, ruleID)
			}
		}
		parts := []string{}
		if result.Errors > 0 {
			parts = append(parts, cp.Error(fmt.Sprintf("%d errors", result.Errors)))
		}
		if result.Warnings > 0 {
			parts = append(parts, cp.Warning(fmt.Sprintf("%d warnings", result.Warnings)))
		}
		if result.Infos > 0 {
			parts = append(parts, cp.Info(fmt.Sprintf("%d infos", result.Infos)))
		}
		if len(parts) == 0 {
			fmt.Println(cp.Bold("No issues found."))
		} else {
			fmt.Printf("\n%s\n", strings.Join(parts, ", "))
		}
	}

	if result.Errors > 0 {
		return 1
	}
	return 0
}
