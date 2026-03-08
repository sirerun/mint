package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirerun/mint/internal/deploy"
)

// stringSliceFlag collects multiple flag values into a slice.
type stringSliceFlag []string

func (f *stringSliceFlag) String() string { return strings.Join(*f, ", ") }
func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

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

	project := fs.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "GCP project ID")
	region := fs.String("region", "us-central1", "GCP region")
	source := fs.String("source", "", "Path to generated server directory")
	serviceName := fs.String("service", "", "Cloud Run service name (default: derived from source dir)")
	imageTag := fs.String("image-tag", "latest", "Container image tag")
	public := fs.Bool("public", false, "Allow unauthenticated access")
	canary := fs.Int("canary", 0, "Traffic percentage for canary (0 = full rollout)")
	vpc := fs.String("vpc", "", "VPC connector name")
	waf := fs.Bool("waf", false, "Enable Cloud Armor")
	internal := fs.Bool("internal", false, "Internal-only ingress")
	kmsKey := fs.String("kms-key", "", "CMEK encryption key")
	timeout := fs.Int("timeout", 300, "Request timeout in seconds")
	maxInstances := fs.Int("max-instances", 10, "Maximum number of instances")
	minInstances := fs.Int("min-instances", 0, "Minimum number of instances")
	ci := fs.Bool("ci", false, "Generate CI workflow")
	promote := fs.Bool("promote", false, "Promote canary to 100%")
	cpuAlways := fs.Bool("cpu-always", false, "Allocate CPU when idle (for SSE)")
	debugImage := fs.Bool("debug-image", false, "Use alpine base for debugging")
	noSourceRepo := fs.Bool("no-source-repo", false, "Skip Cloud Source Repositories push")

	var secrets stringSliceFlag
	fs.Var(&secrets, "secret", "Secret mapping in ENV_VAR=secret-name format (repeatable)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	// Parse secret mappings.
	var secretMappings []deploy.SecretMapping
	for _, s := range secrets {
		m, err := deploy.ParseSecretFlag(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		secretMappings = append(secretMappings, m)
	}

	config := deploy.DeployConfig{
		ProjectID:    *project,
		Region:       *region,
		SourceDir:    *source,
		ServiceName:  *serviceName,
		ImageTag:     *imageTag,
		Public:       *public,
		Canary:       *canary,
		VPC:          *vpc,
		WAF:          *waf,
		Internal:     *internal,
		KMSKey:       *kmsKey,
		Timeout:      *timeout,
		MaxInstances: *maxInstances,
		MinInstances: *minInstances,
		Secrets:      secretMappings,
		CI:           *ci,
		Promote:      *promote,
		CPUAlways:    *cpuAlways,
		DebugImage:   *debugImage,
		NoSourceRepo: *noSourceRepo,
	}

	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(os.Stderr, "deploy gcp: not yet implemented")
	return 1
}

func runDeployStatus(args []string) int {
	fs := flag.NewFlagSet("mint deploy status", flag.ContinueOnError)
	fs.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "GCP project ID")
	fs.String("region", "us-central1", "GCP region")
	fs.String("service", "", "Cloud Run service name (required)")
	fs.String("format", "", "Output format (json)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	fmt.Fprintln(os.Stderr, "deploy status: not yet implemented")
	return 1
}

func runDeployRollback(args []string) int {
	fs := flag.NewFlagSet("mint deploy rollback", flag.ContinueOnError)
	fs.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "GCP project ID")
	fs.String("region", "us-central1", "GCP region")
	fs.String("service", "", "Cloud Run service name (required)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	fmt.Fprintln(os.Stderr, "deploy rollback: not yet implemented")
	return 1
}

func printDeployUsage() {
	fmt.Print(`mint deploy - Deploy generated MCP servers.

Usage:
  mint deploy <target> [flags]

Targets:
  gcp         Deploy to Google Cloud Run

Flags for 'gcp':
  --project <id>         GCP project ID (or set GOOGLE_CLOUD_PROJECT)
  --region <region>      GCP region (default: us-central1)
  --source <dir>         Path to generated server directory (required)
  --service <name>       Cloud Run service name (default: derived from source dir)
  --image-tag <tag>      Container image tag (default: latest)
  --public               Allow unauthenticated access
  --canary <percent>     Traffic percentage for canary (0 = full rollout)
  --vpc <connector>      VPC connector name
  --waf                  Enable Cloud Armor
  --internal             Internal-only ingress
  --kms-key <key>        CMEK encryption key
  --timeout <seconds>    Request timeout in seconds (default: 300)
  --max-instances <n>    Maximum number of instances (default: 10)
  --min-instances <n>    Minimum number of instances (default: 0)
  --secret <mapping>     Secret mapping ENV_VAR=secret-name (repeatable)
  --ci                   Generate CI workflow
  --promote              Promote canary to 100%
  --cpu-always           Allocate CPU when idle (for SSE)
  --debug-image          Use alpine base for debugging
  --no-source-repo       Skip Cloud Source Repositories push

Run 'mint deploy gcp --help' for more information.
`)
}
