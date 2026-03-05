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
  version     Print the version
  help        Show this help message

Run 'mint <command> --help' for more information on a command.
`)
}
