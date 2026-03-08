package main

import (
	"flag"
	"fmt"
	"os"
)

func runDeploy(args []string) int {
	if len(args) == 0 {
		printDeployUsage()
		return 0
	}

	switch args[0] {
	case "gcp":
		return runDeployGCP(args[1:])
	case "status":
		return runDeployStatus(args[1:])
	case "rollback":
		return runDeployRollback(args[1:])
	case "help", "-h", "--help":
		printDeployUsage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown deploy subcommand: %s\n\nRun 'mint deploy help' for usage.\n", args[0])
		return 1
	}
}

func runDeployGCP(args []string) int {
	fs := flag.NewFlagSet("mint deploy gcp", flag.ContinueOnError)
	fs.String("project", "", "GCP project ID (required)")
	fs.String("region", "us-central1", "GCP region")
	fs.String("source", "", "Path to generated server directory (required)")
	fs.String("service", "", "Cloud Run service name (default: from spec title)")
	fs.Bool("public", false, "Allow unauthenticated access")
	fs.Int("canary", 0, "Canary traffic percentage (1-99, 0 means full rollout)")
	fs.String("vpc", "", "VPC connector name")
	fs.Bool("waf", false, "Enable Cloud Armor WAF")
	fs.Bool("internal", false, "Internal-only ingress")
	fs.String("kms-key", "", "KMS key for encryption")
	fs.Int("timeout", 300, "Cloud Run request timeout in seconds")
	fs.Int("max-instances", 10, "Maximum number of instances")
	fs.Int("min-instances", 0, "Minimum number of instances")
	fs.String("secret", "", "Secret mapping ENV_VAR=secret-name (repeatable)")
	fs.Bool("ci", false, "Generate GitHub Actions workflow and configure Workload Identity Federation")
	fs.Bool("promote", false, "Promote canary revision to 100% traffic")
	fs.Bool("cpu-always", false, "Allocate CPU even when idle (for SSE transport)")
	fs.Bool("debug-image", false, "Use alpine base image with shell for debugging")
	fs.Bool("no-source-repo", false, "Skip pushing source to Cloud Source Repositories")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	fmt.Fprintln(os.Stderr, "deploy gcp: not yet implemented")
	return 1
}

func runDeployStatus(args []string) int {
	fs := flag.NewFlagSet("mint deploy status", flag.ContinueOnError)
	fs.String("project", "", "GCP project ID (required)")
	fs.String("region", "us-central1", "GCP region")
	fs.String("service", "", "Cloud Run service name (required)")
	fs.String("format", "", "Output format (json)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	fmt.Fprintln(os.Stderr, "deploy status: not yet implemented")
	return 1
}

func runDeployRollback(args []string) int {
	fs := flag.NewFlagSet("mint deploy rollback", flag.ContinueOnError)
	fs.String("project", "", "GCP project ID (required)")
	fs.String("region", "us-central1", "GCP region")
	fs.String("service", "", "Cloud Run service name (required)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	fmt.Fprintln(os.Stderr, "deploy rollback: not yet implemented")
	return 1
}

func printDeployUsage() {
	fmt.Print(`mint deploy - Deploy MCP servers to cloud platforms.

Usage:
  mint deploy <subcommand> [flags]

Subcommands:
  gcp         Deploy to Google Cloud Run
  status      Check deployment status
  rollback    Roll back to a previous revision

Run 'mint deploy <subcommand> --help' for more information.
`)
}
