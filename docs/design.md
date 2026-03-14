# Mint Design Document

## Overview

Mint is an open-source Go CLI that turns any OpenAPI 3.0/3.1 specification into a working Go MCP server. It also provides foundational OpenAPI tooling (validation, linting, diffing, merging, transformation, overlay) needed to prepare specs for generation.

The primary deliverable is: `mint mcp generate spec.yaml --output ./server` produces a deployable MCP server written in Go.

Repository: `github.com/sirerun/mint`
License: Apache 2.0

## Architecture

### Directory Structure

```
cmd/mint/           CLI entry point with flag-based subcommand dispatch
internal/
  mcpgen/           OpenAPI-to-MCP model converter (model.go, converter)
  mcpgen/golang/    Go code generation with embedded templates
  loader/           OpenAPI spec loading (file, URL, stdin) via libopenapi
  validate/         Structural spec validation
  lint/             Linting with configurable rulesets (minimal/recommended/strict)
  diff/             Spec diffing with breaking change detection
  merge/            Spec merging with conflict strategies
  overlay/          OpenAPI Overlay application
  transform/        Spec transformation (filter, cleanup, format, Swagger 2.0 convert)
  color/            Terminal color utilities
  deploy/           Deploy configuration model, validation, shared domain validation
  deploy/gcp/       GCP Cloud Run deployment (interfaces, business logic, adapters)
  deploy/aws/       AWS ECS Fargate deployment (interfaces, business logic, adapters)
  deploy/azure/     Azure Container Apps deployment (interfaces, business logic, adapters)
  deploy/managed/   Managed hosting API client (deploy, status, list, delete)
  registry/         MCP server registry (index, search, list, install)
templates/mcp-go/   Reference copies of Go templates
examples/           Petstore, merge, overlay examples
testdata/           Test OpenAPI specs
```

### MCP Server Generation Pipeline

**Stage 1: Parse and Resolve**
- Load OpenAPI spec via `pb33f/libopenapi`
- Resolve all `$ref` references
- Validate the spec (optional, enabled by default)

**Stage 2: Map to MCP Model**
- Each OpenAPI operation becomes one MCP tool
- Tool name: operationId converted to snake_case (or derived from method+path)
- Tool description: operation summary or description
- Tool inputSchema: JSON Schema object combining path params, query params, and request body
- Auth config: derived from OpenAPI securitySchemes

**Stage 3: Generate Go Code**
- Execute Go templates against the MCP model
- Write generated files to output directory
- Generated files: main.go, server.go, tools.go, client.go, types.go, go.mod, Dockerfile, README.md

### Deploy Architecture

The deploy feature uses a three-layer interface-based dependency injection architecture:

```
Deployer (orchestrator)
  -> Bridge Adapters (business logic: ensure-if-missing, retry, poll)
    -> SDK Adapters (thin wrappers around cloud SDK calls)
```

Each layer is separated by Go interfaces, making everything testable with mocks. SDK adapters extract narrow interfaces for the underlying cloud SDK methods so unit tests never call real cloud APIs.

**Shared components:**
- `internal/deploy/config.go` -- DeployConfig struct, validation, flag parsing.
- `cmd/mint/deploy.go` -- CLI flag parsing, adapter instantiation, and orchestration calls for all providers and subcommands (aws, gcp, status, rollback).

**GCP Cloud Run deployment:**
- `internal/deploy/gcp/deploy.go` -- Deployer orchestrator with 8 pluggable interfaces.
- `internal/deploy/gcp/*.go` -- Interface definitions and business logic for each concern (registry, build, cloudrun, iam, secrets, sourcerepo, sourcepush, status, rollback, canary, healthcheck, workloadidentity, labels, workflow).
- `internal/deploy/gcp/*_adapter.go` -- Concrete GCP SDK adapter implementations.
- `internal/deploy/gcp/adapters.go` -- Bridge adapter layer connecting SDK clients to Deployer interfaces.

**AWS ECS Fargate deployment:**
- `internal/deploy/aws/deploy.go` -- Deployer struct with 6 orchestrator interfaces: RegistryProvisioner, ImageBuilder, ServiceDeployer, IAMConfigurator, SecretProvisioner, HealthProber.
- `internal/deploy/aws/*.go` -- Interface definitions: ECRClient, CodeBuildClient, ECSClient, ALBClient, IAMClient, SecretsClient, StatusClient, RollbackClient, CanaryClient, OIDCClient.
- `internal/deploy/aws/*_adapter.go` -- SDK adapters with extracted narrow SDK interfaces (ecsAPI, elbv2API, ecrAPI, codebuildAPI, iamAPI, secretsManagerAPI) for unit testing.
- `internal/deploy/aws/adapters.go` -- Bridge adapters (registryBridge, buildBridge, ecsBridge, iamBridge, secretsBridge, healthBridge).
- `internal/deploy/aws/auth.go` -- AWS authentication via SDK default chain + STS GetCallerIdentity.
- `internal/deploy/aws/apis.go` -- Service prerequisite check (7 AWS services).
- `internal/deploy/aws/status.go`, `rollback.go`, `canary.go` -- Status, rollback, canary traffic splitting.
- `internal/deploy/aws/workflow.go`, `oidc.go` -- CI workflow generation and OIDC identity provider setup.
- `internal/deploy/aws/healthcheck.go` -- HTTP health probe with exponential backoff.

**Azure Container Apps deployment:**
- `internal/deploy/azure/deploy.go` -- Deployer struct with 5 orchestrator interfaces: ACRClient, ContainerAppClient, ManagedEnvironmentClient, KeyVaultClient, RBACClient.
- `internal/deploy/azure/*.go` -- Interface definitions for each Azure service.
- `internal/deploy/azure/*_adapter.go` -- SDK adapters using `github.com/Azure/azure-sdk-for-go/sdk`.
- `internal/deploy/azure/adapters.go` -- Bridge adapter layer.
- `internal/deploy/azure/auth.go` -- Azure credential resolution via `azidentity.NewDefaultAzureCredential()`.
- `internal/deploy/azure/status.go`, `rollback.go`, `canary.go` -- Container Apps revision-based traffic splitting (same model as Cloud Run).
- `internal/deploy/azure/workflow.go`, `oidc.go` -- GitHub Actions workflow with OIDC federated identity.
- `internal/deploy/azure/autoscale.go` -- KEDA scale rules for HTTP concurrent requests.
- `internal/deploy/azure/observability.go` -- Log Analytics workspace configuration.
- `internal/deploy/azure/domain.go` -- Custom domain with managed certificate.
- Decision rationale: docs/adr/008-azure-container-apps-deployment-target.md.

**Managed hosting:**
- `internal/deploy/managed/client.go` -- HostingClient interface (Deploy, Status, Delete, ListServers). HTTP client targeting `api.mintmcp.com/v1/hosting`.- `internal/deploy/managed/upload.go` -- CreateSourceTarball + multipart upload with progress.
- `internal/deploy/managed/auth.go` -- LoadToken from env/file, SaveToken. `mint login` command.
- `internal/deploy/managed/deploy.go` -- DeployFromSource with polling (exponential backoff).
- `internal/deploy/managed/format.go` -- FormatStatus, FormatServerList (table/JSON).
- Decision rationale: docs/adr/009-managed-mcp-hosting-platform.md.

**MCP server registry:**
- `internal/registry/types.go` -- RegistryEntry (Name, Description, Tags, SpecURL, AuthType, AuthEnvVar, MinMintVersion), RegistryIndex.
- `internal/registry/index.go` -- FetchIndex from GitHub raw URL, cached at `~/.cache/mint/registry.json` with 1-hour TTL, offline fallback.
- `internal/registry/search.go` -- Fuzzy search with scoring (name=1.0, contains=0.8, tag=0.6, description=0.4).
- `internal/registry/list.go` -- List with tag filtering.
- `internal/registry/install.go` -- Downloads OpenAPI spec from SpecURL, prints post-install instructions for `mint mcp generate`.
- Registry data: `sirerun/mcp-registry` GitHub repo with 20 curated API entries.
- Decision rationale: docs/adr/010-mcp-server-registry.md (includes relationship to official MCP registry).

**Shared deploy components:**
- `internal/deploy/domain.go` -- ValidateDomain function used by all providers for `--domain` flag.

**Production hardening (all providers):**
- Auto-scaling: GCP (Cloud Run native), AWS (Application Auto Scaling with CPU target tracking), Azure (KEDA HTTP concurrent requests).
- Custom domains: `--domain` flag with managed TLS on all providers.
- Graceful shutdown: Generated servers handle SIGTERM/SIGINT with configurable timeout.
- Observability: GCP (Cloud Logging labels), AWS (CloudWatch Logs + Container Insights), Azure (Log Analytics).

**Adapter architecture pattern (all providers):**
- CloudRunAdapter (GCP) is split into 4 sub-adapter structs because Go does not allow methods with the same name but different return types on one struct. See docs/adr/005-gcp-sdk-adapter-pattern.md.
- AWS adapters extract narrow interfaces (e.g., `ecsAPI`, `elbv2API`) for the underlying SDK clients, enabling unit testing of adapter methods without real AWS credentials.
- Azure adapters follow the same narrow-interface pattern as AWS.

### Key Dependencies

| Dependency | Purpose | Used In |
|-----------|---------|---------|
| pb33f/libopenapi | OpenAPI parsing | mint binary |
| mark3labs/mcp-go | Go MCP SDK | Generated servers only |
| cloud.google.com/go/* | GCP SDK (multiple services) | mint binary (GCP adapters) |
| github.com/aws/aws-sdk-go-v2/* | AWS SDK v2 (multiple services) | mint binary (AWS adapters) |
| github.com/Azure/azure-sdk-for-go/sdk/* | Azure SDK (multiple services) | mint binary (Azure adapters) |

### AWS Service Mapping

| GCP Service | AWS Equivalent | Go SDK Module |
|-------------|---------------|---------------|
| Artifact Registry | ECR | `aws-sdk-go-v2/service/ecr` |
| Cloud Build | CodeBuild | `aws-sdk-go-v2/service/codebuild` |
| Cloud Run | ECS Fargate | `aws-sdk-go-v2/service/ecs` |
| (ALB for traffic) | Elastic Load Balancing v2 | `aws-sdk-go-v2/service/elasticloadbalancingv2` |
| Secret Manager | Secrets Manager | `aws-sdk-go-v2/service/secretsmanager` |
| IAM | IAM | `aws-sdk-go-v2/service/iam` |

### Azure Service Mapping

| GCP Service | Azure Equivalent | Azure SDK Module |
|-------------|-----------------|-----------------|
| Artifact Registry | Azure Container Registry | `azcontainerregistry` |
| Cloud Build | ACR Tasks | `azcontainerregistry` |
| Cloud Run | Container Apps | `armappcontainers` |
| Cloud Run traffic split | Container Apps revisions | `armappcontainers` |
| Secret Manager | Key Vault | `azsecrets` |
| IAM | Azure RBAC | `armauthorization` |
| Workload Identity | Federated Identity Credential | `armauthorization` |

### CI/CD

- `go-semantic-release` on merge to main: analyzes conventional commits, creates git tag + GitHub Release.
- `goreleaser` builds binaries for linux/darwin/windows (amd64/arm64), publishes to Homebrew tap.
- CI workflow runs: go build, go test -race, go vet, golangci-lint on every PR.

## Conventions

- CLI built with standard `flag` package, not cobra/viper. Subcommand dispatch via switch statement.
- Testing with standard `testing` package. No testify. Table-driven tests.
- golangci-lint v2 config requires `version: "2"` and `formatters:` section.
- Pre-commit hook runs golangci-lint on packages (not files) + go test.
- libopenapi `Parameter.Required` is `*bool` -- use derefBool helper.
- libopenapi `BuildV3Model()` returns `(*DocumentModel, error)` not `[]error`.
- golangci-lint v2: `gofmt` is a formatter not a linter, `gosimple` merged into staticcheck.
- .gitignore uses `/mint` not `mint` to avoid matching cmd/mint directory.
- Examples dir has its own go.mod -- excluded from lint hook.

### Definition of Done

A task is done when:
1. Code compiles with zero warnings.
2. All new code has unit tests that pass.
3. `go test ./...` passes with no regressions.
4. `golangci-lint run` passes with no new findings.
5. `gofmt -s` produces no changes.
6. Adapter satisfies its interface (compile-time `var _ Interface = (*Adapter)(nil)` check).

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Never allow changes to pile up. Make many small logical commits.
- Each commit should represent one logical change and have a clear message.
- Conventional Commits format: feat(scope):, fix(scope):, docs:, test:, chore:.

## Key File Paths

| Path | Description |
|------|-------------|
| cmd/mint/main.go | CLI entry point and subcommand dispatch |
| cmd/mint/mcp.go | `mint mcp generate` command |
| cmd/mint/deploy.go | Deploy CLI dispatch (aws, azure, gcp, managed, status, rollback) |
| cmd/mint/deploy_managed.go | Managed hosting CLI (deploy, status, list, delete) |
| cmd/mint/registry.go | Registry CLI (search, list, install) |
| internal/deploy/config.go | DeployConfig, validation, SecretMapping |
| internal/deploy/domain.go | ValidateDomain shared by all providers |
| internal/deploy/gcp/deploy.go | GCP Deployer orchestrator (8 interface deps) |
| internal/deploy/gcp/adapters.go | GCP bridge adapters |
| internal/deploy/aws/deploy.go | AWS Deployer orchestrator (6 interface deps) |
| internal/deploy/aws/adapters.go | AWS bridge adapters |
| internal/deploy/azure/deploy.go | Azure Deployer orchestrator (5 interface deps) |
| internal/deploy/azure/adapters.go | Azure bridge adapters |
| internal/deploy/managed/client.go | Managed hosting API client |
| internal/registry/index.go | Registry index fetch + cache |
| internal/mcpgen/model.go | MCP model structs |
| internal/mcpgen/converter.go | OpenAPI-to-MCP model converter |
| internal/mcpgen/golang/generate.go | Go code generation orchestrator |
| .goreleaser.yml | Cross-platform release config |
| .github/workflows/release.yml | go-semantic-release + goreleaser |
| .github/workflows/ci.yml | Build, test, vet, lint |

## Completed Milestones

| Milestone | Date | Description |
|-----------|------|-------------|
| M1: Foundation | 2026-03 | CLI compiles, loads specs, CI green |
| M2: Core OpenAPI Tools | 2026-03 | Lint, diff, merge, overlay, transform commands work |
| M3: MCP Generation | 2026-03 | `mint mcp generate` produces working Go MCP servers |
| M4: MCP Advanced + CI/CD | 2026-03 | Auth, SSE, filtering, GitHub Actions |
| M5: Ship It | 2026-03 | README, examples, v0.1.0 release |
| M6-M10: Deploy Scaffold | 2026-03 | Interface design, business logic, mock tests for deploy |
| M11: Adapters Complete | 2026-03 | All 8 GCP SDK adapter files compile, interface checks pass |
| M12: CLI Wired | 2026-03 | deploy gcp, status, rollback execute real GCP calls |
| M13: Production Ready | 2026-03 | Manual e2e validation passes with Twitter API v2 spec |
| M14: AWS Scaffold | 2026-03 | AWS Deployer struct, interfaces, orchestration with unit tests |
| M15: AWS Adapters Complete | 2026-03 | All 6 AWS SDK adapters + bridge layer |
| M16: AWS Status/Rollback/Canary | 2026-03 | Status, rollback, canary business logic with unit tests |
| M17: AWS CLI Wired | 2026-03 | deploy aws, status --provider aws, rollback --provider aws |
| M18: AWS CI Wired | 2026-03 | Workflow generation, OIDC setup, --ci and --promote flags |
| M19: Azure Scaffold | 2026-03 | Azure Deployer compiles, orchestration tested with mocks |
| M20: Azure Adapters Complete | 2026-03 | All 5 Azure SDK adapters + bridge layer, 100% coverage |
| M21: Azure CLI Wired | 2026-03 | deploy azure, status, rollback, canary, CI workflow generation |
| M22: Managed Hosting Client | 2026-03 | deploy managed sends to API, returns URL |
| M23: Registry Live | 2026-03 | registry search/install works with 20 curated APIs |

## References

- MCP Specification: https://modelcontextprotocol.io
- Official MCP Registry: https://registry.modelcontextprotocol.io (pre-built servers; complementary to mint registry)
- OpenAPI Specification: https://spec.openapis.org/oas/v3.1.0
- pb33f/libopenapi: https://github.com/pb33f/libopenapi
- mark3labs/mcp-go: https://github.com/mark3labs/mcp-go
- Mint Registry: https://github.com/sirerun/mcp-registry (OpenAPI spec catalog for mint mcp generate)
