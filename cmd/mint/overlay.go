package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/overlay"
)

func runOverlay(args []string) int {
	if len(args) == 0 {
		printOverlayUsage()
		return 0
	}

	switch args[0] {
	case "apply":
		return runOverlayApply(args[1:])
	case "help", "-h", "--help":
		printOverlayUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown overlay subcommand: %s\n", args[0])
		return 1
	}
}

func runOverlayApply(args []string) int {
	fs := flag.NewFlagSet("mint overlay apply", flag.ContinueOnError)
	output := fs.String("o", "", "Output file (default: stdout)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "error: spec and overlay files required\n\nUsage: mint overlay apply [flags] <spec> <overlay>")
		return 1
	}

	specData, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading spec: %v\n", err)
		return 1
	}

	overlayData, err := os.ReadFile(fs.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading overlay: %v\n", err)
		return 1
	}

	doc, err := overlay.Parse(overlayData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing overlay: %v\n", err)
		return 1
	}

	result, err := overlay.Apply(specData, doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error applying overlay: %v\n", err)
		return 1
	}

	if *output != "" {
		if err := os.WriteFile(*output, result, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			return 1
		}
	} else {
		fmt.Print(string(result))
	}

	return 0
}

func printOverlayUsage() {
	fmt.Print(`mint overlay - Apply OpenAPI Overlay documents.

Usage:
  mint overlay <subcommand> [flags]

Subcommands:
  apply    Apply an overlay to a spec

Usage:
  mint overlay apply [flags] <spec-file> <overlay-file>

Flags:
  -o <file>    Output file (default: stdout)
`)
}
