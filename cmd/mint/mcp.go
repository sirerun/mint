package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/loader"
	"github.com/sirerun/mint/internal/mcpgen"
	"github.com/sirerun/mint/internal/mcpgen/golang"
)

func runMCP(args []string) int {
	if len(args) == 0 {
		printMCPUsage()
		return 0
	}

	switch args[0] {
	case "generate":
		return runMCPGenerate(args[1:])
	case "help", "-h", "--help":
		printMCPUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown mcp subcommand: %s\n\nRun 'mint mcp help' for usage.\n", args[0])
		return 1
	}
}

func runMCPGenerate(args []string) int {
	fs := flag.NewFlagSet("mint mcp generate", flag.ContinueOnError)
	output := fs.String("output", "./server", "Output directory for generated server")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: spec file path required\n\nUsage: mint mcp generate [flags] <spec-file>")
		return 1
	}

	specPath := fs.Arg(0)

	result, err := loader.Load(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading spec: %v\n", err)
		return 1
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error converting spec: %v\n", err)
		return 1
	}

	if err := golang.Generate(server, *output); err != nil {
		fmt.Fprintf(os.Stderr, "error generating server: %v\n", err)
		return 1
	}

	fmt.Printf("MCP server generated in %s\n", *output)
	fmt.Printf("  Tools: %d\n", len(server.Tools))
	fmt.Println("\nNext steps:")
	fmt.Printf("  cd %s && go mod tidy && go build -o %s .\n", *output, server.Name)

	return 0
}

func printMCPUsage() {
	fmt.Print(`mint mcp - MCP server generation commands.

Usage:
  mint mcp <subcommand> [flags]

Subcommands:
  generate    Generate a Go MCP server from an OpenAPI spec

Run 'mint mcp generate --help' for more information.
`)
}
