package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirerun/mint/internal/deploy/managed"
)

func runDeployManaged(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "status":
			return runDeployManagedStatus(args[1:])
		case "list":
			return runDeployManagedList(args[1:])
		case "delete":
			return runDeployManagedDelete(args[1:])
		}
	}

	fs := flag.NewFlagSet("mint deploy managed", flag.ContinueOnError)
	source := fs.String("source", "", "Path to generated server directory (required)")
	serviceName := fs.String("service", "", "Service name (default: derived from source dir)")
	public := fs.Bool("public", false, "Allow public access")
	apiURL := fs.String("api-url", "", "Hosting API base URL (or set MINT_API_URL)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *source == "" {
		fmt.Fprintln(os.Stderr, "error: --source is required")
		return 1
	}

	// Derive service name from source directory if not provided.
	name := *serviceName
	if name == "" {
		name = filepath.Base(*source)
	}

	// Resolve API URL.
	baseURL := *apiURL
	if baseURL == "" {
		baseURL = os.Getenv("MINT_API_URL")
	}

	token, err := managed.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	client := managed.NewClient(baseURL, token)
	ctx := context.Background()

	result, err := managed.DeployFromSource(ctx, client, *source, name, *public, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Println(result.URL)
	return 0
}

func runDeployManagedStatus(args []string) int {
	fs := flag.NewFlagSet("mint deploy managed status", flag.ContinueOnError)
	serviceName := fs.String("service", "", "Service name (required)")
	apiURL := fs.String("api-url", "", "Hosting API base URL (or set MINT_API_URL)")
	format := fs.String("format", "", "Output format (json)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *serviceName == "" {
		fmt.Fprintln(os.Stderr, "error: --service is required")
		return 1
	}

	baseURL := *apiURL
	if baseURL == "" {
		baseURL = os.Getenv("MINT_API_URL")
	}

	token, err := managed.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	client := managed.NewClient(baseURL, token)
	ctx := context.Background()

	status, err := client.Status(ctx, *serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Print(managed.FormatStatus(status, *format == "json"))
	return 0
}

func runDeployManagedList(args []string) int {
	fs := flag.NewFlagSet("mint deploy managed list", flag.ContinueOnError)
	apiURL := fs.String("api-url", "", "Hosting API base URL (or set MINT_API_URL)")
	format := fs.String("format", "", "Output format (json)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	baseURL := *apiURL
	if baseURL == "" {
		baseURL = os.Getenv("MINT_API_URL")
	}

	token, err := managed.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	client := managed.NewClient(baseURL, token)
	ctx := context.Background()

	servers, err := client.ListServers(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Print(managed.FormatServerList(servers, *format == "json"))
	return 0
}

func runDeployManagedDelete(args []string) int {
	fs := flag.NewFlagSet("mint deploy managed delete", flag.ContinueOnError)
	serviceName := fs.String("service", "", "Service name (required)")
	apiURL := fs.String("api-url", "", "Hosting API base URL (or set MINT_API_URL)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *serviceName == "" {
		fmt.Fprintln(os.Stderr, "error: --service is required")
		return 1
	}

	baseURL := *apiURL
	if baseURL == "" {
		baseURL = os.Getenv("MINT_API_URL")
	}

	token, err := managed.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	client := managed.NewClient(baseURL, token)
	ctx := context.Background()

	if err := client.Delete(ctx, *serviceName); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "Service %q deleted.\n", *serviceName)
	return 0
}
