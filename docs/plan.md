# Mint Deploy -- Production-Ready GCP Deployment

## Context

See docs/design.md for full project context, architecture, and conventions.

### Problem Statement

The `mint deploy gcp`, `mint deploy status`, and `mint deploy rollback` commands are scaffolded but not functional. The codebase contains well-designed interfaces, orchestration logic, and comprehensive mock-based tests, but no concrete GCP SDK adapter implementations exist. The CLI entry point (`cmd/mint/deploy.go`) prints "not yet implemented" and exits.

This plan covers implementing the concrete GCP SDK adapters, wiring them into the CLI, and validating end-to-end functionality.

### Current State

The following are COMPLETE and tested (with mocks):
- `internal/deploy/config.go` -- DeployConfig, validation, SecretMapping parsing.
- `internal/deploy/gcp/deploy.go` -- Deployer orchestrator with 8 interface deps.
- `internal/deploy/gcp/registry.go` -- EnsureRepository logic (uses AR protobuf types).
- `internal/deploy/gcp/build.go` -- BuildImage logic (delegates to BuildClient interface).
- `internal/deploy/gcp/cloudrun.go` -- EnsureService logic (delegates to CloudRunClient interface).
- `internal/deploy/gcp/iam.go` -- ConfigureIAMPolicy logic (delegates to IAMPolicyClient interface).
- `internal/deploy/gcp/secrets.go` -- EnsureSecrets logic (delegates to SecretClient interface).
- `internal/deploy/gcp/sourcerepo.go` -- EnsureSourceRepo logic (delegates to SourceRepoClient interface).
- `internal/deploy/gcp/sourcepush.go` -- PushSource logic (delegates to GitClient interface).
- `internal/deploy/gcp/status.go` -- GetStatus and FormatStatus logic (delegates to StatusClient interface).
- `internal/deploy/gcp/rollback.go` -- Rollback logic (delegates to RevisionClient interface).
- `internal/deploy/gcp/canary.go` -- SetCanaryTraffic and PromoteCanary logic (delegates to TrafficClient interface).
- `internal/deploy/gcp/healthcheck.go` -- HealthChecker with retries (real HTTP implementation).
- `internal/deploy/gcp/labels.go` -- Label generation and sanitization.
- `internal/deploy/gcp/workflow.go` -- GitHub Actions workflow generation.
- `internal/deploy/gcp/workloadidentity.go` -- EnsureWorkloadIdentity logic (delegates to IAMClient interface).
- `internal/deploy/gcp/auth.go` -- GCP Application Default Credentials.
- `cmd/mint/deploy.go` -- Flag parsing and config construction (COMPLETE). Orchestration call (STUBBED).

The following are NOT implemented:
- Concrete GCP SDK adapter structs implementing the interfaces.
- CLI wiring to instantiate adapters and call the Deployer.
- GCP SDK dependencies in go.mod for Cloud Run, Cloud Build, Secret Manager, IAM, Source Repos.
- GCP API enablement check.
- Integration tests with real GCP calls (optional, not required for production readiness).

### Objectives

1. Implement concrete GCP SDK adapters for all 8 interface types.
2. Wire `cmd/mint/deploy.go` to instantiate adapters and call the Deployer orchestrator.
3. Wire `mint deploy status` to call GetStatus with a real StatusClient adapter.
4. Wire `mint deploy rollback` to call Rollback with a real RevisionClient adapter.
5. Add GCP API enablement check before deployment.
6. Validate end-to-end with the petstore example against a real GCP project.

### Non-Goals

- Changing existing interface definitions or business logic.
- Adding new deploy targets (AWS, Azure).
- Custom domain or SSL management.
- Monitoring dashboards.

### Constraints and Assumptions

- GCP Go SDK packages follow the `cloud.google.com/go/<service>/apiv2` convention.
- Cloud Run Admin API v2 (`cloud.google.com/go/run/apiv2`) is the target SDK.
- Cloud Build API (`cloud.google.com/go/cloudbuild/apiv1/v2`) for container builds.
- Secret Manager API (`cloud.google.com/go/secretmanager/apiv1`) for secrets.
- IAM Admin API (`cloud.google.com/go/iam/admin/apiv1`) for service accounts.
- Source Repo API is deprecated; Cloud Source Repositories push should be optional and off by default.
- Git operations use `os/exec` to shell out to `git` binary. Decision rationale: docs/adr/005-gcp-sdk-adapter-pattern.md.

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| `mint deploy gcp` works end-to-end | Deploys petstore MCP server to Cloud Run | Manual test against real GCP project |
| `mint deploy status` returns service info | Shows revisions, URL, traffic split | Manual test |
| `mint deploy rollback` reverts traffic | Shifts traffic to previous revision | Manual test |
| All existing tests pass | Zero regressions | `go test ./...` |
| Build succeeds | `go build ./cmd/mint/` | CI |

---

## Scope and Deliverables

### In Scope

1. **GCP SDK adapter implementations** -- One adapter file per GCP service.
2. **CLI wiring** -- Replace "not yet implemented" stubs with real orchestration calls.
3. **GCP API enablement check** -- Verify required APIs are enabled before deploying.
4. **go.mod dependency additions** -- Add missing GCP SDK packages.
5. **End-to-end validation** -- Manual test with real GCP project.

### Out of Scope

- New features beyond what the interfaces already define.
- Automated integration tests requiring GCP credentials in CI.
- Changes to existing interface definitions.
- AWS/Azure support.

### Deliverables Table

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D20 | GCP SDK adapter implementations | TBD | All 8 interfaces have concrete implementations using real GCP SDK clients |
| D21 | CLI wiring for deploy, status, rollback | TBD | All three commands execute real GCP operations instead of printing "not yet implemented" |
| D22 | GCP API enablement check | TBD | Clear error message listing missing APIs when required APIs are not enabled |

---

## Checkable Work Breakdown

### Epic E20: GCP SDK Adapter Implementations

Decision rationale: docs/adr/005-gcp-sdk-adapter-pattern.md.

- [x] T20.1 Add GCP SDK dependencies to go.mod  Owner: TBD  Est: 15m
  - Run `go get` for:
    - `cloud.google.com/go/run/apiv2`
    - `cloud.google.com/go/cloudbuild/apiv1/v2`
    - `cloud.google.com/go/secretmanager/apiv1`
    - `cloud.google.com/go/iam/admin/apiv1`
  - Run `go mod tidy`.
  - Acceptance: `go build ./cmd/mint/` succeeds. No unused deps.
  - Deps: none

- [x] T20.2 Implement Artifact Registry adapter (`registry_adapter.go`)  Owner: TBD  Est: 45m
  - Create `internal/deploy/gcp/registry_adapter.go`.
  - Struct `ArtifactRegistryAdapter` wrapping `artifactregistry.Client`.
  - Constructor `NewArtifactRegistryAdapter(ctx context.Context) (*ArtifactRegistryAdapter, error)` that creates a real SDK client.
  - Implement `RegistryClient` interface methods: `GetRepository`, `CreateRepository`.
  - Both methods delegate directly to the underlying SDK client, translating between the interface types and SDK types.
  - Acceptance: Compiles. Satisfies `RegistryClient` interface (compile-time check via `var _ RegistryClient = (*ArtifactRegistryAdapter)(nil)`).
  - Deps: T20.1
  - [x] S20.2.1 Add compile-time interface check and unit test  Est: 15m
  - [x] S20.2.2 Run linter and formatter  Est: 10m

- [x] T20.3 Implement Cloud Build adapter (`build_adapter.go`)  Owner: TBD  Est: 1h
  - Create `internal/deploy/gcp/build_adapter.go`.
  - Struct `CloudBuildAdapter` wrapping `cloudbuild.Client`.
  - Implement `BuildClient` interface: `CreateBuild`.
  - CreateBuild must:
    1. Create a tar.gz of the source directory.
    2. Upload the tarball to a Cloud Storage bucket (auto-created by Cloud Build).
    3. Submit a build request with the Dockerfile.
    4. Poll for completion using the returned long-running operation.
    5. Return BuildResult with image URI, log URL, duration, status.
  - Acceptance: Compiles. Satisfies `BuildClient` interface.
  - Deps: T20.1
  - [x] S20.3.1 Add compile-time interface check and unit test  Est: 15m
  - [x] S20.3.2 Run linter and formatter  Est: 10m

- [x] T20.4 Implement Cloud Run adapter (`cloudrun_adapter.go`)  Owner: TBD  Est: 1h
  - Create `internal/deploy/gcp/cloudrun_adapter.go`.
  - Struct `CloudRunAdapter` wrapping `run.ServicesClient`.
  - Implement `CloudRunClient` interface: `GetService`, `CreateService`, `UpdateService`.
  - Map between internal `ServiceConfig`/`Service` types and Cloud Run SDK protobuf types (`runpb.Service`, `runpb.RevisionTemplate`, etc.).
  - GetService: return `ErrNotFound` when gRPC status is `codes.NotFound`.
  - CreateService/UpdateService: wait for the long-running operation to complete, then extract the service URL and revision name from the result.
  - Also implement `StatusClient` interface: `GetService` (returning `ServiceStatus`), `ListRevisions`.
  - Also implement `RevisionClient` interface: `ListRevisions` (returning `[]Revision`), `UpdateTraffic`.
  - Also implement `TrafficClient` interface: `GetTraffic`, `SetTraffic`.
  - These can be on the same adapter struct since they all use the Cloud Run API.
  - Acceptance: Compiles. Satisfies `CloudRunClient`, `StatusClient`, `RevisionClient`, and `TrafficClient` interfaces.
  - Deps: T20.1
  - [x] S20.4.1 Add compile-time interface checks and unit test  Est: 20m
  - [x] S20.4.2 Run linter and formatter  Est: 10m

- [x] T20.5 Implement IAM adapter (`iam_adapter.go`)  Owner: TBD  Est: 45m
  - Create `internal/deploy/gcp/iam_adapter.go`.
  - Struct `IAMAdapter`.
  - For `IAMPolicyClient` interface: use `run.ServicesClient.GetIamPolicy` and `SetIamPolicy` (IAM on Cloud Run services is accessed via the Run API, not a separate IAM API).
  - Map between internal `IAMPolicy`/`IAMBinding` types and `iampb.Policy`/`iampb.Binding`.
  - For `IAMClient` interface (used by workload identity): use `iam/admin/apiv1` to create/get service accounts.
  - Acceptance: Compiles. Satisfies `IAMPolicyClient` and `IAMClient` interfaces.
  - Deps: T20.1, T20.4 (shares Cloud Run client for IAM policy on services)
  - [x] S20.5.1 Add compile-time interface checks and unit test  Est: 15m
  - [x] S20.5.2 Run linter and formatter  Est: 10m

- [x] T20.6 Implement Secret Manager adapter (`secrets_adapter.go`)  Owner: TBD  Est: 45m
  - Create `internal/deploy/gcp/secrets_adapter.go`.
  - Struct `SecretManagerAdapter` wrapping `secretmanager.Client`.
  - Implement `SecretClient` interface: `GetSecret`, `CreateSecret`.
  - GetSecret: return `NotFoundErr` when gRPC status is `codes.NotFound`.
  - CreateSecret: create with automatic replication policy.
  - Acceptance: Compiles. Satisfies `SecretClient` interface.
  - Deps: T20.1
  - [x] S20.6.1 Add compile-time interface check and unit test  Est: 15m
  - [x] S20.6.2 Run linter and formatter  Est: 10m

- [x] T20.7 Implement Source Repository adapter (`sourcerepo_adapter.go`)  Owner: TBD  Est: 30m
  - Create `internal/deploy/gcp/sourcerepo_adapter.go`.
  - Struct `SourceRepoAdapter`.
  - Implement `SourceRepoClient` interface: `GetRepo`, `CreateRepo`.
  - Note: Cloud Source Repositories API is being deprecated. This adapter is optional and used only when `--no-source-repo` is not set. Log a deprecation warning.
  - Use `google.golang.org/api/sourcerepo/v1` (REST API) since there is no gRPC client library.
  - Acceptance: Compiles. Satisfies `SourceRepoClient` interface.
  - Deps: T20.1
  - [x] S20.7.1 Add compile-time interface check and unit test  Est: 15m
  - [x] S20.7.2 Run linter and formatter  Est: 10m

- [x] T20.8 Implement Git adapter (`git_adapter.go`)  Owner: TBD  Est: 30m
  - Create `internal/deploy/gcp/git_adapter.go`.
  - Struct `ExecGitClient` that shells out to the `git` binary via `os/exec`.
  - Implement `GitClient` interface: `Init`, `AddAll`, `Commit`, `AddRemote`, `Push`, `HasRemote`.
  - Each method runs `git <subcommand>` in the specified directory.
  - Check that `git` is in PATH; return clear error if not found.
  - Acceptance: Compiles. Satisfies `GitClient` interface. `git` commands execute correctly.
  - Deps: none
  - [x] S20.8.1 Add unit test (mock exec or test with temp dir)  Est: 15m
  - [x] S20.8.2 Run linter and formatter  Est: 10m

### Epic E21: CLI Wiring

- [x] T21.1 Wire `runDeployGCP` to call the Deployer orchestrator  Owner: TBD  Est: 1.5h
  - In `cmd/mint/deploy.go`, replace the "not yet implemented" stub in `runDeployGCP` with:
    1. Call `gcp.Authenticate(ctx)` to get default credentials.
    2. Instantiate all adapter structs (ArtifactRegistryAdapter, CloudBuildAdapter, CloudRunAdapter, IAMAdapter, SecretManagerAdapter, SourceRepoAdapter if enabled, ExecGitClient).
    3. Create a `gcp.Deployer` with all adapters injected.
    4. Construct `gcp.DeployInput` from the `deploy.DeployConfig`.
    5. Call `deployer.Deploy(ctx, input)`.
    6. Print the result (service URL to stdout, details to stderr).
    7. Handle canary flow: if `--canary` > 0, call `gcp.SetCanaryTraffic` after deploy. If `--promote`, call `gcp.PromoteCanary`.
    8. Handle `--ci` flow: call `gcp.EnsureWorkloadIdentity` and generate workflow file.
  - Acceptance: `mint deploy gcp --project P --source ./server` executes the full deploy flow. Returns 0 on success.
  - Deps: T20.2, T20.3, T20.4, T20.5, T20.6, T20.7, T20.8
  - [x] S21.1.1 Add unit test for CLI wiring (mock adapters via build tags or constructor injection)  Est: 30m
  - [x] S21.1.2 Run linter and formatter  Est: 10m

- [x] T21.2 Wire `runDeployStatus` to call GetStatus  Owner: TBD  Est: 45m
  - In `cmd/mint/deploy.go`, replace the "not yet implemented" stub in `runDeployStatus` with:
    1. Parse `--project`, `--region`, `--service`, `--format` flags (already defined but not wired).
    2. Authenticate with GCP.
    3. Create `CloudRunAdapter` (which implements `StatusClient`).
    4. Call `gcp.GetStatus(ctx, adapter, projectID, region, serviceName)`.
    5. Format output with `gcp.FormatStatus(result, format == "json")`.
    6. Print to stdout.
  - Acceptance: `mint deploy status --project P --service S` prints service status. `--format json` outputs valid JSON.
  - Deps: T20.4
  - [x] S21.2.1 Add unit test for status CLI path  Est: 15m
  - [x] S21.2.2 Run linter and formatter  Est: 10m

- [x] T21.3 Wire `runDeployRollback` to call Rollback  Owner: TBD  Est: 45m
  - In `cmd/mint/deploy.go`, replace the "not yet implemented" stub in `runDeployRollback` with:
    1. Parse `--project`, `--region`, `--service` flags (already defined but not wired).
    2. Authenticate with GCP.
    3. Create `CloudRunAdapter` (which implements `RevisionClient`).
    4. Call `gcp.Rollback(ctx, adapter, projectID, region, serviceName)`.
    5. Print result: which revision traffic was shifted to.
  - Acceptance: `mint deploy rollback --project P --service S` shifts traffic to previous revision. Returns 0 on success.
  - Deps: T20.4
  - [x] S21.3.1 Add unit test for rollback CLI path  Est: 15m
  - [x] S21.3.2 Run linter and formatter  Est: 10m

### Epic E22: GCP API Enablement Check

- [x] T22.1 Implement API enablement verification  Owner: TBD  Est: 45m
  - Create `internal/deploy/gcp/apis.go`.
  - Use `google.golang.org/api/serviceusage/v1` to check if required APIs are enabled.
  - Required APIs: `run.googleapis.com`, `cloudbuild.googleapis.com`, `artifactregistry.googleapis.com`, `secretmanager.googleapis.com`, `iam.googleapis.com`.
  - If any API is not enabled, print a clear error listing the missing APIs and the `gcloud services enable` command to enable them.
  - Call this check at the start of `runDeployGCP` before instantiating adapters.
  - Acceptance: Missing APIs produce actionable error message. Enabled APIs pass silently.
  - Deps: T20.1
  - [x] S22.1.1 Add unit test with mock serviceusage client  Est: 20m
  - [x] S22.1.2 Run linter and formatter  Est: 10m

### Epic E23: Validation and Cleanup

- [x] T23.1 Run full test suite and fix regressions  Owner: TBD  Est: 30m
  - Run `go test ./...` and fix any failures.
  - Acceptance: All tests pass. Zero regressions.
  - Deps: T21.1, T21.2, T21.3, T22.1

- [x] T23.2 Run linter and formatter on all changed packages  Owner: TBD  Est: 15m
  - `golangci-lint run ./internal/deploy/...`
  - `golangci-lint run ./cmd/mint/...`
  - `gofmt -s -w .`
  - Acceptance: Zero lint findings. Zero formatting changes.
  - Deps: T23.1

- [ ] T23.3 Manual end-to-end validation with petstore example  Owner: TBD  Est: 1h
  - Generate petstore MCP server: `mint mcp generate testdata/petstore.yaml --output /tmp/petstore-mcp`
  - Deploy: `mint deploy gcp --project <test-project> --source /tmp/petstore-mcp --public`
  - Check status: `mint deploy status --project <test-project> --service petstore-mcp`
  - Verify health endpoint: `curl <service-url>/health`
  - Rollback: `mint deploy rollback --project <test-project> --service petstore-mcp`
  - Acceptance: All commands succeed. Health check returns 200. Rollback shifts traffic.
  - Deps: T23.2

---

## Parallel Work

### Track A: Adapters with no cross-deps (can all run in parallel)

Tasks: T20.1 (first), then T20.2, T20.3, T20.6, T20.7, T20.8 (all parallel after T20.1)

### Track B: Cloud Run adapter (needed by IAM, status, rollback)

Tasks: T20.4, then T20.5

### Track C: CLI wiring (depends on adapters)

Tasks: T21.1, T21.2, T21.3 (all parallel after Track A and B complete)

### Track D: API check (independent)

Tasks: T22.1 (parallel with Track A/B)

### Parallel Execution

| Phase | Track A | Track B | Track C | Track D |
|-------|---------|---------|---------|---------|
| Phase 1 | T20.1 | -- | -- | -- |
| Phase 2 | T20.2, T20.3, T20.6, T20.7, T20.8 | T20.4 | -- | T22.1 |
| Phase 3 | -- | T20.5 | -- | -- |
| Phase 4 | -- | -- | T21.1, T21.2, T21.3 | -- |
| Phase 5 | -- | -- | T23.1, T23.2, T23.3 | -- |

**Sync points:**
- T20.1 must complete before all other T20.x tasks.
- T20.4 must complete before T20.5, T21.2, T21.3.
- All T20.x tasks must complete before T21.1.
- All T21.x tasks must complete before T23.x tasks.

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M11: Adapters Complete | E20 | None | All 8 adapter files compile. Interface checks pass. `go build ./cmd/mint/` succeeds. |
| M12: CLI Wired | E21 | M11 | `mint deploy gcp`, `mint deploy status`, `mint deploy rollback` execute real GCP calls (no "not yet implemented"). |
| M13: Production Ready | E22, E23 | M12 | API check works. All tests pass. Lint clean. Manual e2e validation passes. |

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R16 | Cloud Run SDK v2 has different API surface than expected by existing interfaces | High | Medium | Read SDK docs before coding. Adjust adapter mapping layer. Do not change interfaces. |
| R17 | Cloud Build requires Cloud Storage bucket for source upload | Medium | High | Use the Cloud Build SDK's built-in source upload mechanism. Check if gs://[project]_cloudbuild bucket exists. |
| R18 | Cloud Source Repositories API deprecated | Low | High | Default `--no-source-repo` to true. Log deprecation warning. Keep adapter minimal. |
| R19 | Long-running operations (LRO) have inconsistent wait patterns across GCP SDKs | Medium | Medium | Each SDK has its own LRO helper (e.g., `op.Wait(ctx)`). Use SDK-native patterns. |
| R20 | GCP SDK auth fails in CI environments | Medium | Low | Support `GOOGLE_APPLICATION_CREDENTIALS` env var. Document Workload Identity Federation for CI. |

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

1. Self-review all changed files before marking a task complete.
2. Run `go test ./...` and verify no regressions.
3. Run `golangci-lint run` and fix any findings.
4. Run `gofmt -s -w .` to ensure formatting.
5. Verify adapter type assertions compile.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Never allow changes to pile up. Make many small logical commits.
- Each commit should represent one logical change and have a clear message.

---

## Progress Log

### 2026-03-11 -- Change Summary (Plan Created)

- Audited deploy codebase. Found all E13-E19 tasks marked complete but actual code is scaffolded with interfaces only and CLI stubs.
- Updated docs/design.md to document the deploy architecture gap.
- Created new plan with 3 epics: E20 (Adapters), E21 (CLI Wiring), E22 (API Check), E23 (Validation).
- Total: 12 tasks, 24 subtasks.
- Created ADR: docs/adr/005-gcp-sdk-adapter-pattern.md -- Adapter file naming and structure.
- Changed Source Repo default to opt-out (--no-source-repo) due to API deprecation (R18).
- 3 milestones defined: M11 through M13.
- 5 risks identified: R16 through R20.

### 2026-03-11 -- Plan Created

- No implementation progress yet. Plan is new.

---

## Hand-off Notes

### What You Need to Know

1. **Architecture**: The deploy feature uses interface-based dependency injection. All business logic is complete and tested with mocks. You are implementing the "adapters" -- thin wrappers around real GCP SDK clients.
2. **Key pattern**: Each adapter file follows `<service>_adapter.go` naming. Each contains a struct wrapping the SDK client and implementing one or more interfaces from the same package.
3. **Existing code to study**: `internal/deploy/gcp/registry.go` shows the interface + business logic pattern. Your adapter must implement `RegistryClient` from that file.
4. **CLI entry point**: `cmd/mint/deploy.go` lines 115-116 is where "not yet implemented" lives. Replace with adapter instantiation and Deployer.Deploy() call.
5. **The Deployer orchestrator**: `internal/deploy/gcp/deploy.go` is the top-level flow. Read it to understand the call sequence.
6. **Testing strategy**: Existing mock tests validate business logic. Adapter tests should be minimal (compile-time interface checks, basic unit tests). Real validation is manual e2e.

### Credentials and Links (Placeholders)

- GCP project for testing: TBD (create a dedicated test project).
- Required IAM roles for deployer: `roles/run.admin`, `roles/artifactregistry.admin`, `roles/cloudbuild.builds.editor`, `roles/secretmanager.admin`, `roles/iam.serviceAccountAdmin`.
- No API keys or secrets stored in repository.

---

## Appendix

### GCP SDK Package References

| Interface | GCP SDK Package | Key Types |
|-----------|----------------|-----------|
| RegistryClient | `cloud.google.com/go/artifactregistry/apiv1` | `artifactregistrypb.Repository`, `artifactregistrypb.CreateRepositoryRequest` |
| BuildClient | `cloud.google.com/go/cloudbuild/apiv1/v2` | `cloudbuildpb.Build`, `cloudbuildpb.CreateBuildRequest` |
| CloudRunClient | `cloud.google.com/go/run/apiv2` | `runpb.Service`, `runpb.CreateServiceRequest`, `runpb.UpdateServiceRequest` |
| StatusClient | `cloud.google.com/go/run/apiv2` | `runpb.Service`, `runpb.Revision`, `runpb.ListRevisionsRequest` |
| RevisionClient | `cloud.google.com/go/run/apiv2` | `runpb.Revision`, `runpb.UpdateServiceRequest` (for traffic) |
| TrafficClient | `cloud.google.com/go/run/apiv2` | `runpb.TrafficTarget`, `runpb.UpdateServiceRequest` |
| IAMPolicyClient | `cloud.google.com/go/run/apiv2` (GetIamPolicy/SetIamPolicy on services) | `iampb.Policy`, `iampb.Binding` |
| IAMClient | `cloud.google.com/go/iam/admin/apiv1` | `adminpb.ServiceAccount`, `adminpb.CreateServiceAccountRequest` |
| SecretClient | `cloud.google.com/go/secretmanager/apiv1` | `secretmanagerpb.Secret`, `secretmanagerpb.CreateSecretRequest` |
| SourceRepoClient | `google.golang.org/api/sourcerepo/v1` | REST API types |
| GitClient | `os/exec` | No SDK, shell out to `git` binary |

### Adapter Constructor Pattern

Each adapter follows this pattern:

```
type CloudRunAdapter struct {
    services *run.ServicesClient
    revisions *run.RevisionsClient
}

func NewCloudRunAdapter(ctx context.Context) (*CloudRunAdapter, error) {
    svc, err := run.NewServicesClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("creating Cloud Run services client: %w", err)
    }
    rev, err := run.NewRevisionsClient(ctx)
    if err != nil {
        svc.Close()
        return nil, fmt.Errorf("creating Cloud Run revisions client: %w", err)
    }
    return &CloudRunAdapter{services: svc, revisions: rev}, nil
}

func (a *CloudRunAdapter) Close() error {
    a.revisions.Close()
    return a.services.Close()
}
```
