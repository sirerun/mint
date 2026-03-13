# Mint Deploy -- AWS ECS Fargate Deployment (Feature Parity with GCP)

## Context

See docs/design.md for full project context, architecture, and conventions.

### Problem Statement

Mint currently supports deploying generated MCP servers to GCP Cloud Run via `mint deploy gcp`. Users on AWS have no equivalent. This plan adds `mint deploy aws` with full feature parity: deploy, status, rollback, canary traffic splitting, secrets, IAM, health checks, and CI workflow generation.

### Objectives

1. Users can run `mint deploy aws --source ./server --region us-east-1` to deploy a generated MCP server to AWS ECS Fargate.
2. `mint deploy status --provider aws` reports service info and task revisions.
3. `mint deploy rollback --provider aws` shifts traffic to the previous task definition revision.
4. Canary deployments split traffic via ALB weighted target groups.
5. Secrets from AWS Secrets Manager are injected as environment variables.
6. IAM controls public vs private access via ALB security groups and IAM task roles.
7. `--ci` generates a GitHub Actions workflow with OIDC-based AWS authentication.

### Non-Goals

- Multi-region or multi-account deployments.
- EKS, App Runner, or Lambda deployment targets.
- Terraform or CloudFormation generation (mint provisions directly via AWS SDK).
- VPC creation from scratch (users provide a VPC ID or mint uses the default VPC).

### Constraints and Assumptions

- AWS Go SDK v2 (`github.com/aws/aws-sdk-go-v2`) for all AWS API calls.
- Same interface-based adapter architecture as GCP (see docs/adr/005-gcp-sdk-adapter-pattern.md).
- AWS credentials resolved via standard SDK chain (env vars, ~/.aws, IRSA, OIDC).
- Default VPC is used unless `--vpc-id` is specified.
- Decision rationale: docs/adr/007-aws-ecs-fargate-deployment-target.md.

### Success Metrics

- `mint deploy aws` deploys a generated MCP server and returns a working URL.
- `mint deploy status --provider aws` shows service and task info.
- `mint deploy rollback --provider aws` shifts traffic to previous revision.
- All new code has unit tests with mock adapters. E2E validation against a real AWS account.

---

## Scope and Deliverables

### In Scope

- `mint deploy aws` subcommand with flags mirroring `mint deploy gcp`.
- `mint deploy status` and `mint deploy rollback` extended to support `--provider aws`.
- AWS SDK adapters for ECR, CodeBuild, ECS, ALB, IAM, Secrets Manager.
- Bridge adapter layer connecting SDK clients to Deployer orchestrator.
- Unit tests for all business logic with mock adapters.
- E2E validation against a real AWS account.
- CI workflow generation for GitHub Actions with OIDC.

### Out of Scope

- AWS App Runner, EKS, or Lambda targets.
- Custom VPC creation or management.
- AWS CloudFormation or CDK output.
- Source repository push (CodeCommit is deprecated).

### Deliverables

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D1 | `mint deploy aws` command | TBD | Deploys container to ECS Fargate, returns service URL |
| D2 | AWS status command | TBD | Shows service info, tasks, and ALB target health |
| D3 | AWS rollback command | TBD | Shifts traffic to previous task definition revision |
| D4 | AWS canary support | TBD | Splits ALB traffic between stable and canary target groups |
| D5 | AWS secrets integration | TBD | Injects Secrets Manager values as container env vars |
| D6 | AWS CI workflow | TBD | Generates GitHub Actions YAML with OIDC auth |

---

## Checkable Work Breakdown

### Epic E24: AWS Deploy Package Scaffold

- [x] T24.1 Create `internal/deploy/aws/` package with Deployer struct and orchestrator interfaces  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: none
  - AC: Package compiles. Deployer struct defined with interface fields matching GCP parity: RegistryProvisioner, ImageBuilder, ServiceDeployer, IAMConfigurator, SecretProvisioner, HealthProber. DeployInput/DeployOutput structs defined.
  - Risk: none

- [x] T24.2 Define low-level SDK client interfaces for each AWS service  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T24.1
  - AC: Interfaces defined in separate files: `ecr.go` (ECRClient), `codebuild.go` (CodeBuildClient), `ecs.go` (ECSClient), `alb.go` (ALBClient), `iam.go` (IAMClient), `secrets.go` (SecretsClient). Each interface has the minimal methods needed.
  - Risk: none

- [x] T24.3 Implement Deployer.Deploy orchestration logic  Owner: agent  Est: 1.5h  Done: 2026-03-13
  - Dependencies: T24.1, T24.2
  - AC: Deploy method calls interfaces in sequence: ensure ECR repo, build image, register ECS task definition, create/update ECS service, configure ALB, configure IAM, inject secrets, run health check. Returns DeployOutput with service URL and task ARN.
  - Risk: none

- [x] T24.4 Add unit tests for Deployer orchestration with mock adapters  Owner: agent  Est: 1h  Done: 2026-03-13  Note: done as part of T24.3
  - Dependencies: T24.3
  - AC: Table-driven tests covering happy path, each step failing, and optional steps (secrets, canary). All tests pass.

- [x] T24.5 Run golangci-lint on internal/deploy/aws/  Owner: agent  Est: 15m  Done: 2026-03-13
  - Dependencies: T24.4
  - AC: Zero lint findings.

### Epic E25: AWS SDK Adapters

- [x] T25.1 ECR adapter (`ecr_adapter.go`)  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Implements ECRClient. Creates repository if not exists, returns repository URI. Compile-time interface check.
  - S25.1.1 Add unit tests for ECR adapter  Owner: TBD  Est: 30m

- [x] T25.2 CodeBuild adapter (`codebuild_adapter.go`)  Owner: agent  Est: 1.5h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Implements CodeBuildClient. Creates CodeBuild project if not exists, starts build from source directory (tar.gz upload to S3), polls until complete, returns image URI.
  - S25.2.1 Add unit tests for CodeBuild adapter  Owner: TBD  Est: 30m

- [x] T25.3 ECS adapter (`ecs_adapter.go`)  Owner: agent  Est: 1.5h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Implements ECSClient. Creates/updates ECS cluster (Fargate), registers task definition, creates/updates ECS service. Split into sub-adapters if needed per ADR 005.
  - S25.3.1 Add unit tests for ECS adapter  Owner: TBD  Est: 45m

- [x] T25.4 ALB adapter (`alb_adapter.go`)  Owner: agent  Est: 1.5h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Implements ALBClient. Creates/reuses ALB, creates target group, registers ECS tasks, configures listener rules. Supports weighted target groups for canary.
  - S25.4.1 Add unit tests for ALB adapter  Owner: TBD  Est: 45m

- [x] T25.5 IAM adapter (`iam_adapter.go`)  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Implements IAMClient. Creates ECS task execution role and task role with required policies. Handles ALB security group for public/private access.
  - S25.5.1 Add unit tests for IAM adapter  Owner: TBD  Est: 30m

- [x] T25.6 Secrets Manager adapter (`secrets_adapter.go`)  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Implements SecretsClient. Ensures secrets exist in AWS Secrets Manager, returns secret ARNs for ECS task definition injection.
  - S25.6.1 Add unit tests for Secrets Manager adapter  Owner: TBD  Est: 30m

- [x] T25.7 Bridge adapter layer (`adapters.go`)  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T25.1 through T25.6
  - AC: Bridge structs connect SDK client interfaces to Deployer orchestrator interfaces. Same pattern as `internal/deploy/gcp/adapters.go`.
  - S25.7.1 Add unit tests for bridge adapters  Owner: TBD  Est: 30m

- [x] T25.8 Authentication helper (`auth.go`)  Owner: agent  Est: 45m  Done: 2026-03-13
  - Dependencies: none
  - AC: Resolves AWS credentials via SDK default chain. Returns config with region and account ID. Validates required permissions.
  - S25.8.1 Add unit tests for auth helper  Owner: TBD  Est: 30m

- [x] T25.9 AWS API prerequisite check (`apis.go`)  Owner: agent  Est: 30m  Done: 2026-03-13
  - Dependencies: T25.8
  - AC: Verifies that required AWS services (ECS, ECR, ELB, CodeBuild, Secrets Manager) are accessible in the target region. Returns clear error messages.
  - S25.9.1 Add unit tests for API check  Owner: TBD  Est: 15m

- [x] T25.10 Run golangci-lint on all adapter files  Owner: agent  Est: 15m  Done: 2026-03-13
  - Dependencies: T25.1 through T25.9
  - AC: Zero lint findings.

### Epic E26: AWS Status, Rollback, and Canary

- [x] T26.1 Status command (`status.go`)  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: StatusClient interface defined. GetStatus retrieves ECS service info (running count, desired count, task definition ARN) and ALB target health. FormatStatus outputs human-readable and JSON formats matching GCP output structure.
  - S26.1.1 Add unit tests for status  Owner: TBD  Est: 30m

- [x] T26.2 Rollback command (`rollback.go`)  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T24.2
  - AC: Lists ECS task definition revisions, updates service to use previous revision, waits for deployment stability. Returns RollbackResult with current and previous revision ARNs.
  - S26.2.1 Add unit tests for rollback  Owner: TBD  Est: 30m

- [x] T26.3 Canary traffic splitting (`canary.go`)  Owner: agent  Est: 1.5h  Done: 2026-03-13
  - Dependencies: T25.4
  - AC: Creates second target group for canary revision, configures ALB listener with weighted forward action. Supports --canary percentage and --promote to shift 100% traffic.
  - S26.3.1 Add unit tests for canary  Owner: TBD  Est: 45m

- [x] T26.4 Health check (`healthcheck.go`)  Owner: agent  Est: 30m  Done: 2026-03-13
  - Dependencies: none
  - AC: Reuses existing HealthChecker from GCP package (HTTP probe). ALB health check configured via ECS task definition.
  - S26.4.1 Add unit tests for health check  Owner: TBD  Est: 15m

- [x] T26.5 Run golangci-lint on E26 files  Owner: agent  Est: 15m  Done: 2026-03-13
  - Dependencies: T26.1 through T26.4
  - AC: Zero lint findings.

### Epic E27: CLI Wiring

- [x] T27.1 Add `aws` subcommand to `cmd/mint/deploy.go`  Owner: agent  Est: 1.5h  Done: 2026-03-13
  - Dependencies: T24.3, T25.7
  - AC: `mint deploy aws` parses flags, instantiates AWS SDK adapters, calls Deployer.Deploy. Flags: --region, --source, --service, --image-tag, --public, --canary, --vpc-id, --timeout, --max-instances, --min-instances, --secret, --ci, --promote, --cpu, --memory, --debug-image.
  - Risk: Flag names should match GCP where semantically equivalent.

- [x] T27.2 Extend status and rollback commands with --provider flag  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T26.1, T26.2
  - AC: `mint deploy status --provider aws --service foo --region us-east-1` works. Default provider remains gcp for backward compatibility. `mint deploy rollback --provider aws` works.

- [x] T27.3 Update deploy help text and usage  Owner: agent  Est: 30m  Done: 2026-03-13  Note: done as part of T27.1
  - Dependencies: T27.1, T27.2
  - AC: `mint deploy help` lists both gcp and aws targets. Each target lists its flags.

- [x] T27.4 Run golangci-lint on cmd/mint/  Owner: agent  Est: 15m  Done: 2026-03-13  Note: only pre-existing errcheck on defer Close() remain
  - Dependencies: T27.1 through T27.3
  - AC: Zero lint findings.

### Epic E28: CI Workflow Generation

- [x] T28.1 Generate GitHub Actions workflow for AWS  Owner: agent  Est: 1h  Done: 2026-03-13
  - Dependencies: T27.1
  - AC: `mint deploy aws --ci` generates `.github/workflows/deploy-aws.yml` with: OIDC authentication to AWS, ECR login, CodeBuild trigger or docker build+push, ECS service update. Uses aws-actions/configure-aws-credentials.
  - S28.1.1 Add unit tests for workflow generation  Owner: TBD  Est: 30m

- [x] T28.2 OIDC identity provider setup (`oidc.go`)  Owner: agent  Est: 45m  Done: 2026-03-13
  - Dependencies: T25.5
  - AC: Creates IAM OIDC provider for GitHub Actions if not exists. Creates IAM role with trust policy for the repo. Returns provider ARN and role ARN.
  - S28.2.1 Add unit tests for OIDC setup  Owner: TBD  Est: 30m

- [x] T28.3 Run golangci-lint on E28 files  Owner: agent  Est: 15m  Done: 2026-03-13
  - Dependencies: T28.1, T28.2
  - AC: Zero lint findings.

### Epic E29: E2E Validation

- [ ] T29.1 Deploy Twitter API v2 MCP server to AWS  Owner: TBD  Est: 2h
  - Dependencies: T27.1
  - AC: Generate MCP server from Twitter API v2 spec. Build container and deploy to ECS Fargate in an AWS sandbox account. `mint deploy status --provider aws` shows service info. `curl /health` returns HTTP 200. `mint deploy rollback --provider aws` shifts traffic.

- [ ] T29.2 Validate canary deployment on AWS  Owner: TBD  Est: 1h
  - Dependencies: T26.3, T29.1
  - AC: Deploy with `--canary 20`, verify ALB routes 20% to new target group. `--promote` shifts to 100%.

- [ ] T29.3 Document any bugs found and fix them  Owner: TBD  Est: 1h
  - Dependencies: T29.1, T29.2
  - AC: All bugs found during E2E fixed and committed.

---

## Parallel Work

| Track | Task/Epic IDs | Description |
|-------|--------------|-------------|
| Track A: Core Scaffold | E24 | Deployer struct, interfaces, orchestration logic |
| Track B: SDK Adapters | E25 (after T24.2) | All 6 AWS service adapters can be built in parallel once interfaces are defined |
| Track C: Status/Rollback/Canary | E26 (after T24.2) | Business logic for status, rollback, canary |
| Track D: CI Workflow | E28 (after T27.1) | Workflow generation depends on CLI wiring |

**Sync Points:**
- T24.2 must complete before Tracks B and C can start.
- E25 and E26 must complete before E27 (CLI wiring).
- E27 must complete before E28 (CI) and E29 (E2E).

Within Track B, all adapter tasks (T25.1 through T25.6) can run in parallel since they implement independent interfaces.

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M14: AWS Scaffold | E24 | M13 (complete) | Deployer struct compiles, orchestration logic has unit tests with mocks |
| M15: Adapters Complete | E25 | M14 | All 6 AWS SDK adapters compile, interface checks pass, unit tests pass |
| M16: Status/Rollback/Canary | E26 | M14 | Status, rollback, and canary business logic implemented with unit tests |
| M17: CLI Wired | E27 | M15, M16 | `mint deploy aws`, `mint deploy status --provider aws`, `mint deploy rollback --provider aws` execute real AWS calls |
| M18: Production Ready | E28, E29 | M17 | E2E validation passes against real AWS account with Twitter API v2 spec |

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R1 | ECS Fargate has more moving parts than Cloud Run (VPC, ALB, target groups, cluster) | High complexity in adapter layer | High | Use default VPC to simplify. Encapsulate complexity in adapters. |
| R2 | ALB provisioning takes 2-5 minutes, slowing deploy experience | Slow first deploy | Medium | Reuse existing ALB if available. Log progress to stderr. |
| R3 | AWS SDK v2 API differences from GCP SDK may require different adapter patterns | Adapter sub-splitting per ADR 005 | Medium | Evaluate during T25.3 and T25.4. Split if needed. |
| R4 | CodeBuild source upload requires S3 bucket management | Additional infrastructure to manage | Medium | Auto-create bucket with lifecycle policy (same pattern as GCP Cloud Build). |
| R5 | Default VPC may not exist in all accounts (some orgs delete it) | Deploy fails with unclear error | Low | Check for VPC existence early. Provide clear error with --vpc-id flag guidance. |

---

## Operating Procedure

### Definition of Done

A task is done when:
1. Code compiles with zero warnings.
2. All new code has unit tests that pass.
3. `go test ./...` passes with no regressions.
4. `golangci-lint run` passes with no new findings.
5. `gofmt -s` produces no changes.
6. Adapter satisfies its interface (compile-time `var _ Interface = (*Adapter)(nil)` check).

### Review and QA Steps

1. Run `go test ./internal/deploy/aws/...` before marking any adapter task complete.
2. Run `golangci-lint run ./internal/deploy/aws/...` after each code change.
3. For CLI wiring tasks, manually test with `go run ./cmd/mint deploy aws --help`.
4. E2E tasks require deploying to a real AWS account and verifying output.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Never allow changes to pile up. Make many small logical commits.

---

## Progress Log

### 2026-03-12 -- Plan Created

Created plan for AWS ECS Fargate deployment with feature parity to GCP Cloud Run. Defined 6 epics (E24-E29), 29 tasks. Created ADR 007 for AWS deployment target decision. Trimmed completed E23 (e2e validation) and M13 milestone from plan; preserved in docs/design.md.

---

## Hand-off Notes

- The GCP deploy implementation in `internal/deploy/gcp/` is the reference architecture. The AWS package should mirror its structure: orchestrator interfaces in `deploy.go`, SDK client interfaces in per-service files, adapter implementations in `*_adapter.go`, bridge layer in `adapters.go`.
- AWS credentials are resolved via the standard AWS SDK v2 default chain. No custom credential handling needed.
- The `deploy.DeployConfig` struct in `internal/deploy/config.go` is currently GCP-specific (e.g., `ProjectID`). It will need to be extended or a parallel AWS config struct created. Evaluate whether to refactor DeployConfig to be provider-agnostic or keep separate config structs per provider.
- AWS Go SDK v2 module: `github.com/aws/aws-sdk-go-v2` with per-service modules like `github.com/aws/aws-sdk-go-v2/service/ecs`.

---

## Appendix

### AWS Service Mapping

| GCP Service | AWS Equivalent | Go SDK Module |
|-------------|---------------|---------------|
| Artifact Registry | ECR | `aws-sdk-go-v2/service/ecr` |
| Cloud Build | CodeBuild | `aws-sdk-go-v2/service/codebuild` |
| Cloud Run | ECS Fargate | `aws-sdk-go-v2/service/ecs` |
| (ALB for traffic) | Elastic Load Balancing v2 | `aws-sdk-go-v2/service/elasticloadbalancingv2` |
| Secret Manager | Secrets Manager | `aws-sdk-go-v2/service/secretsmanager` |
| IAM | IAM | `aws-sdk-go-v2/service/iam` |
| Cloud Source Repos | (skipped -- CodeCommit deprecated) | -- |

### Required AWS IAM Permissions

The deployer needs these managed policies or equivalent custom policies:
- `AmazonECS_FullAccess`
- `AmazonEC2ContainerRegistryFullAccess`
- `AWSCodeBuildAdminAccess`
- `SecretsManagerReadWrite`
- `ElasticLoadBalancingFullAccess`
- `IAMFullAccess` (for creating task roles and OIDC providers)

### CLI Flag Mapping

| GCP Flag | AWS Flag | Notes |
|----------|----------|-------|
| --project | (derived from credentials) | AWS account ID from STS |
| --region | --region | Same semantic |
| --source | --source | Same |
| --service | --service | ECS service name |
| --image-tag | --image-tag | Same |
| --public | --public | ALB security group + no auth |
| --canary | --canary | ALB weighted target groups |
| --vpc | --vpc-id | AWS VPC ID |
| --waf | --waf | AWS WAF (future, out of scope) |
| --internal | --internal | Internal ALB |
| --kms-key | --kms-key | AWS KMS key ARN |
| --timeout | --timeout | ECS stop timeout |
| --max-instances | --max-instances | ECS desired count / auto-scaling max |
| --min-instances | --min-instances | ECS auto-scaling min |
| --secret | --secret | Same format, AWS Secrets Manager |
| --ci | --ci | GitHub Actions with OIDC |
| --promote | --promote | Shift ALB traffic to canary |
| --cpu-always | (not applicable) | ECS Fargate always allocates CPU |
| --no-source-repo | (not applicable) | No source repo feature for AWS |
