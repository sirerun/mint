# Mint -- Phase 2: Multi-Cloud, Managed Hosting, Registry, and Production Hardening

## Context

See docs/design.md for full project context, architecture, and conventions.

### Problem Statement

Mint generates MCP servers from OpenAPI specs and deploys them to GCP Cloud Run and AWS ECS Fargate. To reach broad adoption and $5B ARR by 2031, mint needs: (1) Azure support for enterprise customers, (2) a managed hosting platform for zero-friction deployment and revenue generation, (3) a public registry to remove the "find a spec" barrier, and (4) production hardening of existing deploy providers with real E2E validation.

### Objectives

1. `mint deploy azure` deploys generated MCP servers to Azure Container Apps with full feature parity.
2. `mint deploy managed` deploys to Sire-managed infrastructure with zero cloud setup required.
3. `mint registry search/install` provides a curated catalog of pre-built MCP servers from popular APIs.
4. All three deploy providers (GCP, AWS, Azure) validated end-to-end with real cloud accounts.
5. Production hardening: auto-scaling, custom domains, observability, graceful shutdown.

### Non-Goals

- AKS, App Service, or Azure Functions deployment targets.
- EKS, App Runner, or Lambda deployment targets.
- Multi-region or multi-account deployments.
- Terraform, CloudFormation, ARM template, or Bicep generation.
- Pre-built binary hosting in the registry (generate locally only).

### Constraints and Assumptions

- Azure SDK for Go (`github.com/Azure/azure-sdk-for-go/sdk`) for all Azure API calls.
- Same three-layer adapter architecture as GCP and AWS (see docs/adr/005-gcp-sdk-adapter-pattern.md).
- Managed hosting backend is a separate Sire service; mint CLI is a thin client.
- Registry index is a JSON file in a public GitHub repo (`sirerun/mcp-registry`).
- Azure credentials resolved via standard SDK chain (env vars, Azure CLI, managed identity).

### Success Metrics

- `mint deploy azure` deploys a generated MCP server and returns a working URL.
- `mint deploy managed --source ./server` returns a live `*.mcp.sire.run` URL in under 60 seconds.
- `mint registry search stripe` returns results. `mint registry install stripe` produces a working server.
- E2E validation passes on all three cloud providers with the Twitter API v2 MCP server.

---

## Scope and Deliverables

### In Scope

- Azure Container Apps deployment with full feature parity (canary, rollback, status, secrets, RBAC, CI).
- Managed hosting CLI client (`mint deploy managed`) with API integration.
- MCP server registry CLI (`mint registry search/list/install`) with curated index.
- E2E validation for AWS and Azure against real cloud accounts.
- Production hardening: auto-scaling, custom domains, graceful shutdown, observability hooks.

### Out of Scope

- Managed hosting backend infrastructure (separate repo/team).
- Registry web UI (CLI-only for now).
- Billing integration in mint CLI (handled by Sire platform).
- Azure AKS, App Service, or Functions targets.
- Pre-built binary distribution via registry.

### Deliverables

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D7 | `mint deploy azure` command | TBD | Deploys container to Azure Container Apps, returns URL |
| D8 | Azure status/rollback/canary | TBD | Status shows revisions, rollback shifts traffic, canary splits |
| D9 | Azure CI workflow | TBD | Generates GitHub Actions YAML with OIDC federated identity |
| D10 | `mint deploy managed` command | TBD | Deploys to Sire hosting, returns live URL |
| D11 | `mint registry` commands | TBD | search, list, install work against curated index |
| D12 | E2E validation (all providers) | TBD | Twitter MCP server deploys and responds on GCP, AWS, Azure |
| D13 | Production hardening | TBD | Auto-scaling, custom domains, graceful shutdown |

---

## Checkable Work Breakdown

### Phase D: Azure Container Apps Deployment

#### Epic E30: Azure Deploy Package Scaffold

- [x] T30.1 Create `internal/deploy/azure/` package with Deployer struct and orchestrator interfaces  Owner: TBD  Est: 1h
  - Dependencies: none
  - AC: Package compiles. Deployer struct defined with interface fields: RegistryProvisioner, ImageBuilder, ServiceDeployer, IAMConfigurator, SecretProvisioner, HealthProber. DeployInput/DeployOutput defined.
  - Decision rationale: docs/adr/008-azure-container-apps-deployment-target.md

- [x] T30.2 Define SDK client interfaces for Azure services  Owner: TBD  Est: 1h
  - Dependencies: T30.1
  - AC: Interfaces defined: ACRClient (Azure Container Registry), ContainerAppClient, ManagedEnvironmentClient, KeyVaultClient, RBACClient. Each in a separate file.

- [x] T30.3 Implement Deployer.Deploy orchestration logic  Owner: TBD  Est: 1.5h
  - Dependencies: T30.1, T30.2
  - AC: Deploy method calls interfaces in sequence: ensure ACR, build image, ensure Container Apps Environment, create/update Container App, configure RBAC, inject secrets, run health check. Returns DeployOutput with URL.

- [x] T30.4 Add unit tests for Deployer orchestration  Owner: TBD  Est: 1h
  - Dependencies: T30.3
  - AC: Table-driven tests covering happy path, each step failing, optional steps. All tests pass with -race.

- [x] T30.5 Run golangci-lint on internal/deploy/azure/  Owner: TBD  Est: 15m
  - Dependencies: T30.4
  - AC: Zero lint findings.

#### Epic E31: Azure SDK Adapters

- [x] T31.1 ACR adapter (`acr_adapter.go`)  Owner: TBD  Est: 1h
  - Dependencies: T30.2
  - AC: Implements ACRClient. Creates ACR repository if not exists, returns login server and image URI. Extracted SDK interface for testing.

- [x] T31.2 Container App adapter (`containerapp_adapter.go`)  Owner: TBD  Est: 1.5h
  - Dependencies: T30.2
  - AC: Implements ContainerAppClient. Creates/updates Container App with revision, configures ingress, sets resource limits (CPU, memory). Extracted SDK interface for testing.

- [x] T31.3 Managed Environment adapter (`environment_adapter.go`)  Owner: TBD  Est: 1h
  - Dependencies: T30.2
  - AC: Implements ManagedEnvironmentClient. Creates Container Apps Environment if not exists. Configures log analytics workspace.

- [x] T31.4 Key Vault adapter (`keyvault_adapter.go`)  Owner: TBD  Est: 1h
  - Dependencies: T30.2
  - AC: Implements KeyVaultClient. Creates Key Vault if not exists, stores secrets, returns secret URIs for container app reference.

- [x] T31.5 RBAC adapter (`rbac_adapter.go`)  Owner: TBD  Est: 1h
  - Dependencies: T30.2
  - AC: Implements RBACClient. Assigns AcrPull role to Container App managed identity. Configures Key Vault access policies.

- [x] T31.6 Authentication helper (`auth.go`)  Owner: TBD  Est: 45m
  - Dependencies: none
  - AC: Resolves Azure credentials via SDK default chain (env, CLI, managed identity). Returns subscription ID, resource group, and tenant ID.

- [x] T31.7 Bridge adapter layer (`adapters.go`)  Owner: TBD  Est: 1h
  - Dependencies: T31.1 through T31.6
  - AC: Bridge structs connect SDK client interfaces to Deployer orchestrator interfaces.

- [x] T31.8 Unit tests for all adapters  Owner: TBD  Est: 2h
  - Dependencies: T31.1 through T31.7
  - AC: 100% coverage on business logic. Extracted SDK interfaces with mock implementations.

- [x] T31.9 Run golangci-lint on all adapter files  Owner: TBD  Est: 15m
  - Dependencies: T31.8
  - AC: Zero lint findings.

#### Epic E32: Azure Status, Rollback, and Canary

- [x] T32.1 Status command (`status.go`)  Owner: TBD  Est: 1h
  - Dependencies: T30.2
  - AC: StatusClient interface defined. GetStatus retrieves Container App info (revisions, traffic weights, replica counts). FormatStatus outputs human-readable and JSON.

- [x] T32.2 Rollback command (`rollback.go`)  Owner: TBD  Est: 1h
  - Dependencies: T30.2
  - AC: Lists Container App revisions, shifts 100% traffic to previous revision.

- [x] T32.3 Canary traffic splitting (`canary.go`)  Owner: TBD  Est: 1h
  - Dependencies: T31.2
  - AC: Creates new revision with canary traffic percentage. PromoteCanary shifts 100% to canary revision. Uses Container Apps native traffic splitting.

- [x] T32.4 Health check (`healthcheck.go`)  Owner: TBD  Est: 30m
  - Dependencies: none
  - AC: HTTP health probe with exponential backoff (same pattern as GCP/AWS).

- [x] T32.5 Unit tests for E32  Owner: TBD  Est: 1.5h
  - Dependencies: T32.1 through T32.4
  - AC: 100% coverage. Table-driven tests with mocks.

- [x] T32.6 Run golangci-lint on E32 files  Owner: TBD  Est: 15m
  - Dependencies: T32.5
  - AC: Zero lint findings.

#### Epic E33: Azure CLI Wiring

- [x] T33.1 Add `azure` subcommand to `cmd/mint/deploy.go`  Owner: TBD  Est: 1.5h
  - Dependencies: T30.3, T31.7
  - AC: `mint deploy azure` parses flags, instantiates Azure SDK adapters, calls Deployer.Deploy. Flags: --subscription, --resource-group, --region, --source, --service, --image-tag, --public, --canary, --timeout, --max-instances, --min-instances, --secret, --ci, --promote, --cpu, --memory.

- [x] T33.2 Extend status and rollback with --provider azure  Owner: TBD  Est: 1h
  - Dependencies: T32.1, T32.2
  - AC: `mint deploy status --provider azure` and `mint deploy rollback --provider azure` work.

- [x] T33.3 Update deploy help text  Owner: TBD  Est: 30m
  - Dependencies: T33.1, T33.2
  - AC: `mint deploy help` lists aws, azure, and gcp targets.

- [x] T33.4 Run golangci-lint on cmd/mint/  Owner: TBD  Est: 15m
  - Dependencies: T33.1 through T33.3
  - AC: Zero lint findings.

#### Epic E34: Azure CI Workflow Generation

- [x] T34.1 Generate GitHub Actions workflow for Azure  Owner: TBD  Est: 1h
  - Dependencies: T33.1
  - AC: `mint deploy azure --ci` generates `.github/workflows/deploy-azure.yml` with OIDC federated identity using azure/login action.

- [x] T34.2 OIDC federated identity setup (`oidc.go`)  Owner: TBD  Est: 45m
  - Dependencies: T31.5
  - AC: Creates Azure AD app registration and federated credential for GitHub Actions. Returns client ID and tenant ID.

- [x] T34.3 Unit tests for E34  Owner: TBD  Est: 45m
  - Dependencies: T34.1, T34.2
  - AC: 100% coverage for workflow generation and OIDC setup.

- [x] T34.4 Run golangci-lint on E34 files  Owner: TBD  Est: 15m
  - Dependencies: T34.3
  - AC: Zero lint findings.

### Phase A: Managed MCP Hosting

#### Epic E35: Managed Hosting CLI Client

- [x] T35.1 Create `internal/deploy/managed/` package with API client  Owner: TBD  Est: 1.5h
  - Dependencies: none
  - AC: Package compiles. HostingClient interface defined with methods: Deploy, Status, Delete, ListServers. HTTP client implementation targeting `api.sire.run/v1/hosting`.
  - Decision rationale: docs/adr/009-managed-mcp-hosting-platform.md

- [x] T35.2 Implement source upload (tarball creation and upload)  Owner: TBD  Est: 1h
  - Dependencies: T35.1
  - AC: Creates tar.gz of source directory, uploads via multipart POST to hosting API. Shows upload progress on stderr.

- [x] T35.3 Implement deploy command with polling  Owner: TBD  Est: 1h
  - Dependencies: T35.2
  - AC: Calls deploy endpoint, polls for build/deploy status, prints progress, returns live URL on success.

- [x] T35.4 Implement status, list, and delete commands  Owner: TBD  Est: 1h
  - Dependencies: T35.1
  - AC: `mint deploy managed status --service foo` shows service info. `mint deploy managed list` shows all servers. `mint deploy managed delete --service foo` removes server.

- [x] T35.5 Authentication via API token  Owner: TBD  Est: 45m
  - Dependencies: T35.1
  - AC: Reads SIRE_API_TOKEN from env or `~/.config/mint/credentials`. `mint login` prompts for token and saves it. Clear error when token is missing.

- [x] T35.6 Unit tests for managed hosting client  Owner: TBD  Est: 1.5h
  - Dependencies: T35.1 through T35.5
  - AC: 100% coverage. Uses httptest.Server for API mock.

- [x] T35.7 Run golangci-lint  Owner: TBD  Est: 15m
  - Dependencies: T35.6
  - AC: Zero lint findings.

#### Epic E36: Managed Hosting CLI Wiring

- [x] T36.1 Add `managed` subcommand to `cmd/mint/deploy.go`  Owner: TBD  Est: 1h
  - Dependencies: T35.3
  - AC: `mint deploy managed --source ./server` uploads, builds, deploys, returns URL. Flags: --source, --service, --public.

- [x] T36.2 Add `mint login` command  Owner: TBD  Est: 45m
  - Dependencies: T35.5
  - AC: `mint login` reads token from stdin or --token flag, saves to credentials file.

- [x] T36.3 Update deploy help text with managed option  Owner: TBD  Est: 15m
  - Dependencies: T36.1
  - AC: `mint deploy help` lists managed alongside aws, azure, gcp.

- [x] T36.4 Run golangci-lint on cmd/mint/  Owner: TBD  Est: 15m
  - Dependencies: T36.1 through T36.3
  - AC: Zero lint findings.

### Phase B: MCP Server Registry

#### Epic E37: Registry Index and CLI

- [x] T37.1 Create `internal/registry/` package with index types  Owner: TBD  Est: 1h
  - Dependencies: none
  - AC: Package compiles. RegistryEntry struct: Name, Description, Tags, SpecURL, AuthType, AuthEnvVar, MinMintVersion. RegistryIndex struct: Version, Entries.
  - Decision rationale: docs/adr/010-mcp-server-registry.md

- [x] T37.2 Implement index fetching and caching  Owner: TBD  Est: 1h
  - Dependencies: T37.1
  - AC: Fetches registry index JSON from GitHub raw URL. Caches locally at `~/.cache/mint/registry.json` with TTL of 1 hour. Falls back to cache when offline.

- [x] T37.3 Implement search command  Owner: TBD  Est: 45m
  - Dependencies: T37.2
  - AC: `mint registry search <query>` fuzzy-matches name, description, and tags. Outputs table with name, description, auth type. Supports `--format json`.

- [x] T37.4 Implement list command  Owner: TBD  Est: 30m
  - Dependencies: T37.2
  - AC: `mint registry list` shows all entries. Supports `--tags <tag>` filter. Supports `--format json`.

- [x] T37.5 Implement install command  Owner: TBD  Est: 1.5h
  - Dependencies: T37.2
  - AC: `mint registry install <name>` fetches spec URL from index, runs `mint mcp generate` with appropriate flags (auth-env, output dir). Prints post-install instructions (set env vars, build, run).

- [x] T37.6 Unit tests for registry package  Owner: TBD  Est: 1.5h
  - Dependencies: T37.1 through T37.5
  - AC: 100% coverage. Uses httptest.Server for index fetch. Tests search ranking, cache expiry, offline fallback.

- [x] T37.7 Run golangci-lint  Owner: TBD  Est: 15m
  - Dependencies: T37.6
  - AC: Zero lint findings.

#### Epic E38: Registry CLI Wiring

- [x] T38.1 Add `registry` subcommand to `cmd/mint/main.go`  Owner: TBD  Est: 1h
  - Dependencies: T37.3, T37.4, T37.5
  - AC: `mint registry search`, `mint registry list`, `mint registry install` work end-to-end.

- [x] T38.2 Update help text  Owner: TBD  Est: 15m
  - Dependencies: T38.1
  - AC: `mint help` lists registry commands.

- [x] T38.3 Run golangci-lint on cmd/mint/  Owner: TBD  Est: 15m
  - Dependencies: T38.1, T38.2
  - AC: Zero lint findings.

#### Epic E39: Seed Registry Index

- [x] T39.1 Create `sirerun/mcp-registry` repo with index schema  Owner: TBD  Est: 1h
  - Dependencies: none
  - AC: Repo exists with registry.json schema, validation CI, and CONTRIBUTING.md for community submissions.

- [x] T39.2 Curate initial 20 API entries  Owner: TBD  Est: 3h
  - Dependencies: T39.1
  - AC: 20 entries with verified OpenAPI spec URLs. Mix of categories: social (Twitter, GitHub), payments (Stripe), messaging (Slack, Discord), AI (OpenAI, Anthropic), cloud (AWS, GCP, Azure), dev tools (Jira, Linear, Notion).

- [x] T39.3 Add CI validation to registry repo  Owner: TBD  Est: 1h
  - Dependencies: T39.1
  - AC: GitHub Actions workflow validates registry.json schema, checks spec URL reachability, runs `mint mcp generate` against each entry to verify compatibility.

### Phase C: E2E Validation and Production Hardening

#### Epic E29: E2E Validation (continued from prior plan)

- [ ] T29.1 Deploy Twitter API v2 MCP server to AWS  Owner: TBD  Est: 2h
  - Dependencies: none (AWS deploy is complete)
  - AC: Generate MCP server from Twitter API v2 spec. Deploy to ECS Fargate in AWS sandbox. `curl /health` returns 200. Status and rollback commands work.

- [ ] T29.2 Validate canary deployment on AWS  Owner: TBD  Est: 1h
  - Dependencies: T29.1
  - AC: Deploy with `--canary 20`, verify ALB routes 20% to new target group. `--promote` shifts to 100%.

- [ ] T29.3 Deploy Twitter API v2 MCP server to Azure  Owner: TBD  Est: 2h
  - Dependencies: E33 (Azure CLI wired)
  - AC: Deploy to Azure Container Apps. Health check passes. Status and rollback work.

- [ ] T29.4 Validate canary deployment on Azure  Owner: TBD  Est: 1h
  - Dependencies: T29.3
  - AC: Deploy with `--canary 20`, verify revision traffic split. Promote shifts to 100%.

- [ ] T29.5 Document and fix all bugs found during E2E  Owner: TBD  Est: 2h
  - Dependencies: T29.1 through T29.4
  - AC: All bugs fixed and committed.

#### Epic E40: Production Hardening

- [x] T40.1 Auto-scaling policies for AWS  Owner: TBD  Est: 1.5h
  - Dependencies: T29.1
  - AC: ECS Service Auto Scaling configured via Application Auto Scaling SDK. Scales between --min-instances and --max-instances based on CPU utilization (70% target). Unit tests with mock.

- [x] T40.2 Auto-scaling policies for Azure  Owner: TBD  Est: 1h
  - Dependencies: T29.3
  - AC: Container Apps KEDA scale rules configured. Scales on HTTP concurrent requests. Unit tests with mock.

- [x] T40.3 Custom domain support  Owner: TBD  Est: 2h
  - Dependencies: none
  - AC: `mint deploy <provider> --domain api.example.com` configures custom domain with managed TLS. Works on GCP (Cloud Run domain mapping), AWS (ALB + ACM certificate), Azure (Container Apps custom domain + managed certificate).

- [x] T40.4 Graceful shutdown handling  Owner: TBD  Est: 1h
  - Dependencies: none
  - AC: Generated servers handle SIGTERM gracefully: drain in-flight SSE connections, close HTTP listener, exit within --timeout seconds. Update Go templates in `templates/mcp-go/`.

- [x] T40.5 Observability hooks  Owner: TBD  Est: 1.5h
  - Dependencies: none
  - AC: `mint deploy <provider> --observability` configures provider-native logging and metrics. GCP: Cloud Logging + Cloud Monitoring. AWS: CloudWatch Logs + Container Insights. Azure: Log Analytics.

- [x] T40.6 Unit tests for hardening features  Owner: TBD  Est: 2h
  - Dependencies: T40.1 through T40.5
  - AC: 100% coverage on new code. All quality gates pass.

- [x] T40.7 Run golangci-lint on all modified packages  Owner: TBD  Est: 15m
  - Dependencies: T40.6
  - AC: Zero lint findings.

---

## Parallel Work

| Track | Task/Epic IDs | Description |
|-------|--------------|-------------|
| Track A: Azure Scaffold + Adapters | E30, E31 | Azure deployer and SDK adapters |
| Track B: Azure Status/Rollback/Canary | E32 (after T30.2) | Business logic for status, rollback, canary |
| Track C: Azure CLI + CI | E33, E34 (after E31, E32) | CLI wiring and workflow generation |
| Track D: Managed Hosting | E35, E36 | Hosting client and CLI (independent of Azure) |
| Track E: Registry | E37, E38, E39 | Registry index, CLI, seed data (independent of Azure) |
| Track F: E2E + Hardening | E29, E40 (after E33 for Azure) | Validation and production features |

**Sync Points:**
- T30.2 must complete before Tracks A adapters and Track B can start.
- E31 and E32 must complete before E33 (Azure CLI wiring).
- E33 must complete before T29.3 (Azure E2E validation).
- Tracks D and E are fully independent of Track A/B/C and can run in parallel from the start.
- Track F's AWS validation (T29.1, T29.2) can start immediately. Azure validation waits for Track C.

**Agent parallelization (up to 5 agents):**

Wave 1 (5 agents): T30.1 + T30.2, T31.1 + T31.2, T31.3 + T31.4, T31.5 + T31.6, T32.1 + T32.2
Wave 2 (5 agents): T30.3 + T30.4 + T30.5, T31.7 + T31.8 + T31.9, T32.3 + T32.4 + T32.5 + T32.6, T35.1 + T35.2 + T35.3, T37.1 + T37.2 + T37.3
Wave 3 (5 agents): T33.1 + T33.2 + T33.3 + T33.4, T34.1 + T34.2 + T34.3 + T34.4, T35.4 + T35.5 + T35.6 + T35.7, T37.4 + T37.5 + T37.6 + T37.7, T36.1 + T36.2 + T36.3 + T36.4
Wave 4 (5 agents): T38.1 + T38.2 + T38.3, T39.1 + T39.2 + T39.3, T29.1 + T29.2, T29.3 + T29.4, T40.1 + T40.2
Wave 5 (4 agents): T29.5, T40.3 + T40.4, T40.5 + T40.6 + T40.7, verification

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M19: Azure Scaffold | E30 | none | Azure Deployer compiles, orchestration tested with mocks |
| M20: Azure Adapters | E31 | M19 | All Azure SDK adapters compile, 100% coverage |
| M21: Azure CLI Wired | E32, E33, E34 | M20 | `mint deploy azure` executes real Azure calls, CI workflow generates |
| M22: Managed Hosting Client | E35, E36 | none | `mint deploy managed` sends to API, returns URL |
| M23: Registry Live | E37, E38, E39 | none | `mint registry search/install` works with 20 curated APIs |
| M24: All Providers Validated | E29 | M21 | E2E passes on AWS and Azure with Twitter MCP server |
| M25: Production Ready | E40 | M24 | Auto-scaling, custom domains, graceful shutdown, observability |

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R6 | Azure Container Apps API is less mature than GCP/AWS equivalents | Adapter implementation may hit SDK gaps | Medium | Check SDK coverage before starting. Fall back to REST API if needed. |
| R7 | Managed hosting API not ready when CLI client is built | CLI cannot be E2E tested | High | Build against API contract (OpenAPI spec). Use httptest for unit tests. E2E validation happens when API is deployed. |
| R8 | Registry spec URLs go stale as APIs update their specs | Broken `mint registry install` experience | Medium | CI job runs weekly to validate all spec URLs. Community PRs for updates. |
| R9 | Three cloud SDK dependency trees bloat binary size | Larger mint binary, longer build times | Low | Use build tags to compile per-provider if needed. Evaluate after Azure SDK is added. |
| R10 | Managed hosting requires trust boundary for user code | Security risk running untrusted containers | High | gVisor sandbox, read-only filesystem, network policy, resource limits. Design in Phase A, implement in hosting backend. |
| R11 | Azure RBAC model differs significantly from GCP IAM and AWS IAM | IAM adapter is complex | Medium | Study Azure RBAC early. Use managed identity for Container Apps to simplify role assignments. |

---

## Operating Procedure

### Definition of Done

A task is done when:
1. Code compiles with zero warnings.
2. All new code has unit tests with 100% coverage.
3. `go test ./... -race` passes with no regressions.
4. `golangci-lint run` passes with no new findings.
5. `gofmt -s` produces no changes.
6. Adapter satisfies its interface (compile-time check).

### Review and QA Steps

1. Run `go test ./internal/deploy/<provider>/... -race -coverprofile=cover.out` before marking any task complete.
2. Run `golangci-lint run ./internal/deploy/<provider>/...` after each code change.
3. For CLI wiring tasks, manually test with `go run ./cmd/mint deploy <provider> --help`.
4. E2E tasks require deploying to a real cloud account and verifying output.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Never allow changes to pile up. Make many small logical commits.
- Conventional Commits: feat(deploy):, fix(deploy):, test(deploy):, docs:.

---

## Progress Log

### 2026-03-13 -- Plan Created

Created Phase 2 plan covering four initiatives: D (Azure Container Apps), A (Managed Hosting), B (MCP Registry), C (E2E + Hardening). Defined 12 epics (E29-E40), 55 tasks. Trimmed completed AWS epics (E24-E28) from plan; knowledge preserved in docs/design.md (milestones M14-M18, AWS deploy architecture, deploy directory structure). Created three ADRs:
- docs/adr/008-azure-container-apps-deployment-target.md -- Azure Container Apps chosen over ACI, AKS, App Service.
- docs/adr/009-managed-mcp-hosting-platform.md -- Managed hosting via Sire API with `mint deploy managed`.
- docs/adr/010-mcp-server-registry.md -- Public registry with CLI-based generate-locally model.

---

## Hand-off Notes

- The AWS deploy in `internal/deploy/aws/` is the most recent reference implementation. It has 100% test coverage and extracted SDK interfaces. Use it as the template for Azure adapters.
- Azure Container Apps uses revision-based traffic splitting (identical model to Cloud Run), which is simpler than AWS's ALB-based approach. The canary adapter should be straightforward.
- Azure SDK for Go uses a different auth pattern than AWS/GCP: `azidentity.NewDefaultAzureCredential()` returns a `TokenCredential` used by all service clients.
- The managed hosting CLI client is a thin HTTP client. The heavy lifting (build, deploy, DNS, metering) happens in the Sire hosting backend (separate repo).
- The registry is a JSON file in a GitHub repo. The CLI fetches it, caches locally, and delegates to `mint mcp generate` for installation. No binary distribution.
- E2E validation for AWS (T29.1, T29.2) is unblocked and can start now. Azure E2E (T29.3, T29.4) requires the Azure CLI wiring to be complete first.

---

## Appendix

### Azure Service Mapping

| GCP Service | AWS Equivalent | Azure Equivalent | Azure SDK Module |
|-------------|---------------|-----------------|-----------------|
| Artifact Registry | ECR | Azure Container Registry | `azcontainerregistry` |
| Cloud Build | CodeBuild | ACR Tasks | `azcontainerregistry` |
| Cloud Run | ECS Fargate | Container Apps | `armappcontainers` |
| Cloud Run traffic split | ALB weighted targets | Container Apps revisions | `armappcontainers` |
| Secret Manager | Secrets Manager | Key Vault | `azsecrets` |
| IAM | IAM | Azure RBAC | `armauthorization` |
| Workload Identity | OIDC Provider | Federated Identity Credential | `armauthorization` |

### Registry Index Schema

```json
{
  "version": 1,
  "entries": [
    {
      "name": "twitter-v2",
      "description": "Twitter/X API v2",
      "tags": ["social", "twitter"],
      "spec_url": "https://api.twitter.com/2/openapi.json",
      "auth_type": "bearer",
      "auth_env_var": "TWITTER_BEARER_TOKEN",
      "min_mint_version": "0.2.0"
    }
  ]
}
```

### Managed Hosting API Contract

```
POST   /v1/hosting/deploy    -- Upload source, start build+deploy
GET    /v1/hosting/servers    -- List user's servers
GET    /v1/hosting/servers/:id -- Get server status
DELETE /v1/hosting/servers/:id -- Delete server
```

Authentication: `Authorization: Bearer <SIRE_API_TOKEN>`
