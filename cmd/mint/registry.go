package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/registry"
)

func runRegistry(args []string) int {
	if len(args) == 0 {
		printRegistryUsage()
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printRegistryUsage()
		return 0
	case "search":
		return runRegistrySearch(args[1:])
	case "list":
		return runRegistryList(args[1:])
	case "install":
		return runRegistryInstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown registry command: %s\n\nRun 'mint registry help' for usage.\n", args[0])
		return 1
	}
}

func runRegistrySearch(args []string) int {
	fs := flag.NewFlagSet("mint registry search", flag.ContinueOnError)
	format := fs.String("format", "table", "Output format (table or json)")
	indexURL := fs.String("index-url", "", "Override registry index URL")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: search query is required")
		fmt.Fprintln(os.Stderr, "\nUsage: mint registry search <query>")
		return 1
	}
	query := fs.Arg(0)

	ctx := context.Background()
	index, err := registry.GetIndex(ctx, registry.IndexOptions{
		IndexURL: *indexURL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	results := registry.Search(index, query)
	fmt.Print(registry.FormatSearchResults(results, *format == "json"))
	return 0
}

func runRegistryList(args []string) int {
	fs := flag.NewFlagSet("mint registry list", flag.ContinueOnError)
	tags := fs.String("tags", "", "Filter by tag")
	format := fs.String("format", "table", "Output format (table or json)")
	indexURL := fs.String("index-url", "", "Override registry index URL")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	ctx := context.Background()
	index, err := registry.GetIndex(ctx, registry.IndexOptions{
		IndexURL: *indexURL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	entries := registry.List(index, *tags)
	fmt.Print(registry.FormatList(entries, *format == "json"))
	return 0
}

func runRegistryInstall(args []string) int {
	fs := flag.NewFlagSet("mint registry install", flag.ContinueOnError)
	output := fs.String("output", "", "Output directory (default: ./<name>)")
	indexURL := fs.String("index-url", "", "Override registry index URL")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: server name is required")
		fmt.Fprintln(os.Stderr, "\nUsage: mint registry install <name>")
		return 1
	}
	name := fs.Arg(0)

	ctx := context.Background()
	index, err := registry.GetIndex(ctx, registry.IndexOptions{
		IndexURL: *indexURL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	err = registry.Install(ctx, index, registry.InstallOptions{
		Name:      name,
		OutputDir: *output,
	}, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func printRegistryUsage() {
	fmt.Print(`Usage:
  mint registry <command> [flags]

Commands:
  search    Search for MCP servers in the registry
  list      List all MCP servers in the registry
  install   Install an MCP server spec from the registry
  help      Show this help message

Run 'mint registry <command> --help' for more information on a command.
`)
}
