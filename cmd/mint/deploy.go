package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirerun/mint/internal/deploy"
	awspkg "github.com/sirerun/mint/internal/deploy/aws"
	"github.com/sirerun/mint/internal/deploy/gcp"
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
	case "aws":
		return runDeployAWS(args[1:])
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

func runDeployAWS(args []string) int {
	fs := flag.NewFlagSet("mint deploy aws", flag.ContinueOnError)

	region := fs.String("region", os.Getenv("AWS_REGION"), "AWS region")
	source := fs.String("source", "", "Path to generated server directory")
	serviceName := fs.String("service", "", "ECS service name (default: derived from source dir)")
	imageTag := fs.String("image-tag", "latest", "Container image tag")
	public := fs.Bool("public", false, "Allow public access via ALB")
	canary := fs.Int("canary", 0, "Traffic percentage for canary (0 = full rollout)")
	vpcID := fs.String("vpc-id", "", "AWS VPC ID (default: default VPC)")
	timeout := fs.Int("timeout", 300, "ECS stop timeout in seconds")
	maxInstances := fs.Int("max-instances", 10, "ECS desired count / auto-scaling max")
	minInstances := fs.Int("min-instances", 0, "ECS auto-scaling min")
	ci := fs.Bool("ci", false, "Generate CI workflow")
	promote := fs.Bool("promote", false, "Promote canary to 100%")
	debugImage := fs.Bool("debug-image", false, "Use alpine base for debugging")
	cpu := fs.String("cpu", "256", "CPU units (256, 512, 1024, 2048, 4096)")
	memory := fs.String("memory", "512", "Memory in MB (512, 1024, 2048, etc.)")

	var secrets stringSliceFlag
	fs.Var(&secrets, "secret", "Secret mapping ENV_VAR=secret-name (repeatable)")

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
		Region:       *region,
		SourceDir:    *source,
		ServiceName:  *serviceName,
		ImageTag:     *imageTag,
		Public:       *public,
		Canary:       *canary,
		VPC:          *vpcID,
		Timeout:      *timeout,
		MaxInstances: *maxInstances,
		MinInstances: *minInstances,
		Secrets:      secretMappings,
		CI:           *ci,
		Promote:      *promote,
		DebugImage:   *debugImage,
		CPU:          *cpu,
		Memory:       *memory,
	}

	ctx := context.Background()

	// Authenticate with AWS and resolve account ID.
	creds, err := awspkg.Authenticate(ctx, *region, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Set ProjectID from AWS account ID so Validate() passes.
	config.ProjectID = creds.AccountID

	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Instantiate SDK adapters.
	ecrAdapter := awspkg.NewECRAdapter(creds.Config)
	codebuildAdapter := awspkg.NewCodeBuildAdapter(creds.Config)
	ecsAdapter := awspkg.NewECSAdapter(creds.Config)
	albAdapter := awspkg.NewALBAdapter(creds.Config)
	iamAdapter := awspkg.NewIAMAdapter(creds.Config)
	secretsAdapter := awspkg.NewSecretsManagerAdapter(creds.Config)

	// Assemble the Deployer with bridge adapters.
	deployer := &awspkg.Deployer{
		Registry: awspkg.NewRegistryBridge(ecrAdapter),
		Builder:  awspkg.NewBuildBridge(codebuildAdapter, config.ServiceName),
		ECS:      awspkg.NewECSBridge(ecsAdapter),
		IAM:      awspkg.NewIAMBridge(iamAdapter),
		Secrets:  awspkg.NewSecretsBridge(secretsAdapter, os.Stderr),
		Health:   awspkg.NewHealthBridge(awspkg.NewHealthChecker(nil)),
		Stderr:   os.Stderr,
	}

	// Run deployment.
	result, err := deployer.Deploy(ctx, awspkg.DeployInput{
		Config:   &config,
		SpecHash: config.ImageTag,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Handle canary traffic management.
	if config.Canary > 0 && result.TaskARN != "" {
		_, canaryErr := awspkg.SetCanaryTraffic(ctx, albAdapter, awspkg.CanaryConfig{
			ServiceName:   config.ServiceName,
			VPCID:         *vpcID,
			CanaryPercent: config.Canary,
		})
		if canaryErr != nil {
			fmt.Fprintf(os.Stderr, "warning: canary traffic split failed: %v\n", canaryErr)
		} else {
			fmt.Fprintf(os.Stderr, "Canary: %d%% traffic routed to new task\n", config.Canary)
		}
	}

	if config.Promote {
		// TODO: PromoteCanary requires load balancer ARN and canary target group ARN
		// which need to be resolved from the service context.
		fmt.Fprintln(os.Stderr, "warning: --promote for AWS requires load balancer context (not yet wired)")
	}

	// Handle CI workflow generation.
	if config.CI {
		// TODO: AWS CI workflow generation and OIDC setup is being implemented by another teammate.
		fmt.Fprintln(os.Stderr, "warning: --ci for AWS is not yet implemented")
	}

	// Print service URL to stdout.
	fmt.Println(result.ServiceURL)

	if !result.Healthy {
		fmt.Fprintln(os.Stderr, "Warning: service health check failed")
	}

	_ = debugImage // reserved for Dockerfile base image selection

	return 0
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

	ctx := context.Background()

	// Check that required GCP APIs are enabled.
	if err := gcp.CheckAPIsEnabled(ctx, config.ProjectID); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Authenticate with GCP.
	creds, err := gcp.Authenticate(ctx, config.ProjectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	config.ProjectID = creds.ProjectID

	// Instantiate SDK adapters.
	registryAdapter, err := gcp.NewArtifactRegistryAdapter(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer registryAdapter.Close()

	buildAdapter, err := gcp.NewCloudBuildAdapterFromContext(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer buildAdapter.Close()

	crAdapter, err := gcp.NewCloudRunAdapter(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer crAdapter.Close()

	iamPolicyAdapter := gcp.NewIAMPolicyAdapter(crAdapter.Service.ServicesClient())
	secretAdapter, err := gcp.NewSecretManagerAdapter(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer secretAdapter.Close()

	// Assemble the Deployer with bridge adapters.
	deployer := &gcp.Deployer{
		Registry: gcp.NewRegistryBridge(registryAdapter),
		Builder:  gcp.NewBuildBridge(buildAdapter, config.ProjectID),
		CloudRun: gcp.NewCloudRunBridge(crAdapter.Service),
		IAM:      gcp.NewIAMBridge(iamPolicyAdapter),
		Secrets:  gcp.NewSecretsBridge(secretAdapter, os.Stderr),
		Health:   gcp.NewHealthBridge(gcp.NewHealthChecker(nil)),
		Stderr:   os.Stderr,
	}

	// Wire source repo and git if not disabled.
	if !config.NoSourceRepo {
		sourceRepoAdapter, srcErr := gcp.NewSourceRepoAdapter(ctx)
		if srcErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", srcErr)
			return 1
		}
		gitClient, gitErr := gcp.NewExecGitClient()
		if gitErr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", gitErr)
			return 1
		}
		deployer.SourceRepo = gcp.NewSourceRepoBridge(sourceRepoAdapter)
		deployer.Git = gcp.NewGitBridge(gitClient)
	}

	// Run deployment.
	result, err := deployer.Deploy(ctx, gcp.DeployInput{
		Config:   &config,
		SpecHash: config.ImageTag,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Handle canary traffic management.
	if config.Canary > 0 && result.RevisionName != "" {
		serviceName := gcp.ServiceFullName(config.ProjectID, config.Region, config.ServiceName)
		_, canaryErr := gcp.SetCanaryTraffic(ctx, crAdapter.Traffic, gcp.CanaryConfig{
			ServiceName:   serviceName,
			NewRevision:   result.RevisionName,
			CanaryPercent: config.Canary,
		})
		if canaryErr != nil {
			fmt.Fprintf(os.Stderr, "warning: canary traffic split failed: %v\n", canaryErr)
		} else {
			fmt.Fprintf(os.Stderr, "Canary: %d%% traffic routed to %s\n", config.Canary, result.RevisionName)
		}
	}

	if config.Promote {
		serviceName := gcp.ServiceFullName(config.ProjectID, config.Region, config.ServiceName)
		if promoteErr := gcp.PromoteCanary(ctx, crAdapter.Traffic, serviceName); promoteErr != nil {
			fmt.Fprintf(os.Stderr, "warning: canary promotion failed: %v\n", promoteErr)
		} else {
			fmt.Fprintln(os.Stderr, "Canary promoted to 100%")
		}
	}

	// Handle CI workflow generation.
	if config.CI {
		iamSAAdapter, iamErr := gcp.NewIAMServiceAccountAdapter(ctx)
		if iamErr != nil {
			fmt.Fprintf(os.Stderr, "warning: IAM adapter creation failed: %v\n", iamErr)
		} else {
			defer iamSAAdapter.Close()
			wiResult, wiErr := gcp.EnsureWorkloadIdentity(ctx, iamSAAdapter, gcp.WorkloadIdentityConfig{
				ProjectID:     config.ProjectID,
				ProjectNumber: "",
			}, os.Stderr)
			if wiErr != nil {
				fmt.Fprintf(os.Stderr, "warning: workload identity setup failed: %v\n", wiErr)
			} else {
				outputDir := filepath.Dir(config.SourceDir)
				wfResult, wfErr := gcp.GenerateWorkflow(gcp.WorkflowConfig{
					ProjectID:                config.ProjectID,
					Region:                   config.Region,
					ServiceName:              config.ServiceName,
					SourceDir:                config.SourceDir,
					WorkloadIdentityProvider: wiResult.ProviderName,
					ServiceAccountEmail:      wiResult.ServiceAccount,
				}, outputDir)
				if wfErr != nil {
					fmt.Fprintf(os.Stderr, "warning: workflow generation failed: %v\n", wfErr)
				} else {
					fmt.Fprintf(os.Stderr, "CI workflow written to %s\n", wfResult.FilePath)
				}
			}
		}
	}

	// Print service URL to stdout.
	fmt.Println(result.ServiceURL)

	if !result.Healthy {
		fmt.Fprintln(os.Stderr, "Warning: service health check failed")
	}

	return 0
}

func runDeployStatus(args []string) int {
	fs := flag.NewFlagSet("mint deploy status", flag.ContinueOnError)
	project := fs.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "GCP project ID")
	region := fs.String("region", "us-central1", "GCP region")
	service := fs.String("service", "", "Cloud Run service name (required)")
	format := fs.String("format", "", "Output format (json)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *service == "" {
		fmt.Fprintln(os.Stderr, "error: --service is required")
		return 1
	}

	ctx := context.Background()

	creds, err := gcp.Authenticate(ctx, *project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	crAdapter, err := gcp.NewCloudRunAdapter(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer crAdapter.Close()

	result, err := gcp.GetStatus(ctx, crAdapter.Status, creds.ProjectID, *region, *service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Print(gcp.FormatStatus(result, *format == "json"))
	return 0
}

func runDeployRollback(args []string) int {
	fs := flag.NewFlagSet("mint deploy rollback", flag.ContinueOnError)
	project := fs.String("project", os.Getenv("GOOGLE_CLOUD_PROJECT"), "GCP project ID")
	region := fs.String("region", "us-central1", "GCP region")
	service := fs.String("service", "", "Cloud Run service name (required)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *service == "" {
		fmt.Fprintln(os.Stderr, "error: --service is required")
		return 1
	}

	ctx := context.Background()

	creds, err := gcp.Authenticate(ctx, *project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	crAdapter, err := gcp.NewCloudRunAdapter(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer crAdapter.Close()

	result, err := gcp.Rollback(ctx, crAdapter.Revision, creds.ProjectID, *region, *service)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "Rolled back: traffic shifted from %s to %s\n", result.CurrentRevision, result.PreviousRevision)
	return 0
}

func printDeployUsage() {
	fmt.Print(`mint deploy - Deploy generated MCP servers.

Usage:
  mint deploy <target> [flags]

Targets:
  aws         Deploy to AWS ECS Fargate
  gcp         Deploy to Google Cloud Run

Flags for 'aws':
  --region <region>      AWS region (or set AWS_REGION)
  --source <dir>         Path to generated server directory (required)
  --service <name>       ECS service name (default: derived from source dir)
  --image-tag <tag>      Container image tag (default: latest)
  --public               Allow public access via ALB
  --canary <percent>     Traffic percentage for canary (0 = full rollout)
  --vpc-id <id>          AWS VPC ID (default: default VPC)
  --timeout <seconds>    ECS stop timeout in seconds (default: 300)
  --max-instances <n>    ECS desired count / auto-scaling max (default: 10)
  --min-instances <n>    ECS auto-scaling min (default: 0)
  --secret <mapping>     Secret mapping ENV_VAR=secret-name (repeatable)
  --ci                   Generate CI workflow
  --promote              Promote canary to 100%%
  --debug-image          Use alpine base for debugging
  --cpu <units>          CPU units (default: 256)
  --memory <mb>          Memory in MB (default: 512)

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
