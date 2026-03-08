package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage()
		return 0
	case "version", "-v", "--version":
		fmt.Println("mint " + version)
		return 0
	case "mcp":
		return runMCP(args[1:])
	case "validate":
		return runValidate(args[1:])
	case "lint":
		return runLint(args[1:])
	case "diff":
		return runDiff(args[1:])
	case "merge":
		return runMerge(args[1:])
	case "overlay":
		return runOverlay(args[1:])
	case "transform":
		return runTransform(args[1:])
	case "deploy":
		return runDeploy(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\nRun 'mint help' for usage.\n", args[0])
		return 1
	}
}

func printUsage() {
	fmt.Print(`mint - Generate MCP servers from OpenAPI specs.

Usage:
  mint <command> [flags]

Commands:
  mcp         MCP server generation commands
  validate    Validate an OpenAPI spec for correctness
  lint        Lint an OpenAPI spec with configurable rulesets
  diff        Compare two OpenAPI specs
  merge       Merge multiple OpenAPI specs
  overlay     Apply OpenAPI Overlay documents
  transform   Transform specs (filter, cleanup, format)
  deploy      Deploy generated MCP servers
  version     Print the version
  help        Show this help message

Run 'mint <command> --help' for more information on a command.
`)
}
