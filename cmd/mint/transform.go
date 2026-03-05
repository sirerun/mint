package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirerun/mint/internal/transform"
)

func runTransform(args []string) int {
	if len(args) == 0 {
		printTransformUsage()
		return 0
	}

	switch args[0] {
	case "filter":
		return runTransformFilter(args[1:])
	case "cleanup":
		return runTransformCleanup(args[1:])
	case "format":
		return runTransformFormat(args[1:])
	case "help", "-h", "--help":
		printTransformUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown transform subcommand: %s\n", args[0])
		return 1
	}
}

func runTransformFilter(args []string) int {
	fs := flag.NewFlagSet("mint transform filter", flag.ContinueOnError)
	output := fs.String("o", "", "Output file (default: stdout)")
	tags := fs.String("tags", "", "Include only operations with these tags (comma-separated)")
	exclude := fs.String("exclude-paths", "", "Exclude paths matching patterns (comma-separated)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: spec file required")
		return 1
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading spec: %v\n", err)
		return 1
	}

	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
	}
	var excludeList []string
	if *exclude != "" {
		excludeList = strings.Split(*exclude, ",")
	}

	result, err := transform.FilterOperations(data, tagList, excludeList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	return writeOutput(result, *output)
}

func runTransformCleanup(args []string) int {
	fs := flag.NewFlagSet("mint transform cleanup", flag.ContinueOnError)
	output := fs.String("o", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: spec file required")
		return 1
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading spec: %v\n", err)
		return 1
	}

	result, err := transform.RemoveUnusedComponents(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	return writeOutput(result, *output)
}

func runTransformFormat(args []string) int {
	fs := flag.NewFlagSet("mint transform format", flag.ContinueOnError)
	output := fs.String("o", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: spec file required")
		return 1
	}

	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading spec: %v\n", err)
		return 1
	}

	result, err := transform.Normalize(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	return writeOutput(result, *output)
}

func writeOutput(data []byte, path string) int {
	if path != "" {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			return 1
		}
	} else {
		fmt.Print(string(data))
	}
	return 0
}

func printTransformUsage() {
	fmt.Print(`mint transform - Transform OpenAPI specs.

Usage:
  mint transform <subcommand> [flags]

Subcommands:
  filter     Filter operations by tags or path patterns
  cleanup    Remove unused components
  format     Normalize and format a spec

Run 'mint transform <subcommand> --help' for more information.
`)
}
