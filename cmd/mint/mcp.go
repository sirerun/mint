package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v4"

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
	includeTags := fs.String("include-tags", "", "Only include operations with these tags (comma-separated)")
	excludePaths := fs.String("exclude-paths", "", "Exclude paths matching this pattern (comma-separated)")
	authHeader := fs.String("auth-header", "", "Custom auth header name (overrides spec)")
	authEnv := fs.String("auth-env", "", "Custom env var for auth token (overrides spec)")
	toolNames := fs.String("tool-names", "", "YAML file mapping original tool names to custom names")

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

	if *includeTags != "" {
		tags := strings.Split(*includeTags, ",")
		server.Tools = mcpgen.FilterByTags(server.Tools, tags, result.Model)
	}

	if *excludePaths != "" {
		patterns := strings.Split(*excludePaths, ",")
		server.Tools = mcpgen.FilterByPaths(server.Tools, patterns)
	}

	if *toolNames != "" {
		mapping, err := loadToolNameMapping(*toolNames)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading tool names: %v\n", err)
			return 1
		}
		server.Tools = mcpgen.RenameTools(server.Tools, mapping)
	}

	if *authHeader != "" && server.Auth != nil {
		server.Auth.HeaderName = *authHeader
	}
	if *authEnv != "" && server.Auth != nil {
		server.Auth.EnvVar = *authEnv
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

func loadToolNameMapping(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	var mapping map[string]string
	if err := yaml.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	return mapping, nil
}

func printMCPUsage() {
	fmt.Print(`mint mcp - MCP server generation commands.

Usage:
  mint mcp <subcommand> [flags]

Subcommands:
  generate    Generate a Go MCP server from an OpenAPI spec

Flags for 'generate':
  --output <dir>         Output directory (default: ./server)
  --include-tags <tags>  Only include operations with these tags (comma-separated)
  --exclude-paths <pat>  Exclude paths matching patterns (comma-separated)
  --auth-header <name>   Custom auth header name
  --auth-env <var>       Custom env var for auth token
  --tool-names <file>    YAML file mapping original tool names to custom names

Run 'mint mcp generate --help' for more information.
`)
}
