# Mint -- Cloud Deployment for MCP Servers

## Context

See docs/design.md for full project context, architecture, and conventions from completed v0.1.0 work.

### Problem Statement

Mint generates production-quality Go MCP servers from OpenAPI specs, but deploying these servers to production requires manual infrastructure setup -- provisioning container registries, configuring Cloud Run services, setting up IAM policies, managing secrets, and wiring CI/CD pipelines. This manual process is slow, error-prone, and incompatible with AI-native development workflows where features should be released at the speed of thought.

The `mint deploy` command will automate the full deployment lifecycle: build a container image, push it to a registry, provision cloud infrastructure, deploy the MCP server, verify health, and roll back on failure -- all in a single command with SOC2-compliant security defaults.

### Objectives

1. Add a `mint deploy gcp` command that deploys a generated MCP server to Google Cloud Run with a single invocation.
2. Enforce SOC2-compliant security controls by default (IAM auth, Secret Manager, audit logs, distroless containers, TLS 1.2+).
3. Use the GCP Go SDK for all provisioning -- no Terraform, no gcloud CLI dependency. Decision rationale: docs/adr/002-go-sdk-over-terraform-for-provisioning.md.
4. Host deployed MCP server source code in Google Cloud Source Repositories for convenience.
5. Optimize the release pipeline for AI-native workflows with sub-minute deploys, automated health checks, and instant rollback. Decision rationale: docs/adr/003-ai-native-release-pipeline.md.
6. Provide a GitHub Actions workflow template for automated deploy-on-push.

### Non-Goals

- AWS or Azure deployment targets (future work, not in this scope).
- Kubernetes (GKE) orchestration -- Cloud Run is the target, not GKE.
- Custom domain configuration (users can do this via GCP console).
- Multi-region deployment (single region for v1).
- MCP server monitoring dashboards (Cloud Logging and Cloud Monitoring are used but no custom dashboards are provisioned).
- Cost optimization features (reserved instances, committed use discounts).

### Constraints and Assumptions

- Users must have a GCP project with billing enabled.
- Users must have the `gcloud` CLI installed for initial authentication only (`gcloud auth application-default login`). Not used for provisioning.
- The GCP Go SDK packages will be added as dependencies to the mint binary.
- Cloud Run supports HTTP/SSE which maps to MCP HTTP/SSE transport.
- Cloud Run request timeout must be extended (up to 3600s) for long-running SSE connections.
- Generated MCP servers already include a Dockerfile template (from E10.5).

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Deploy time (spec to live endpoint) | Under 90 seconds | Timed from `mint deploy gcp` invocation to health check pass |
| Security controls enabled by default | 100% of SOC2-required controls | Audit of deployed Cloud Run service configuration |
| Rollback time | Under 10 seconds | Timed from rollback command to traffic shift |
| Idempotent deploys | Running deploy twice produces identical state | Automated test: deploy, deploy again, verify no drift |
| AI agent deploy success rate | 95%+ | Deploy triggered by AI coding assistant succeeds without human intervention |

---

## Scope and Deliverables

### In Scope

1. **`mint deploy gcp` command** -- Single command to build, push, and deploy an MCP server to Cloud Run.
2. **GCP resource provisioning via Go SDK** -- Artifact Registry repo, Cloud Run service, IAM policies, Secret Manager secrets, Cloud Source Repositories.
3. **SOC2-compliant defaults** -- IAM auth, distroless containers, Secret Manager, audit logs, non-root execution, TLS 1.2+.
4. **Health check verification** -- Post-deploy MCP initialize request to verify the server is functional.
5. **Automated rollback** -- Revert to previous Cloud Run revision on health check failure.
6. **Cloud Source Repositories integration** -- Push generated server source to a GCP-hosted git repository.
7. **GitHub Actions workflow template** -- Automated deploy-on-push workflow.
8. **Canary deployments** -- Optional traffic splitting for gradual rollout.
9. **`mint deploy status`** -- Check deployment status and current revision.
10. **`mint deploy rollback`** -- Manual rollback to a previous revision.

### Out of Scope

- AWS, Azure, or other cloud provider deployment.
- Kubernetes/GKE deployment.
- Custom domain and SSL certificate management.
- Multi-region or multi-cluster deployment.
- Cost management and billing alerts.
- Monitoring dashboards.
- Deployment approval workflows (manual gates).

### Deliverables Table

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D10 | `mint deploy gcp` command | TBD | Deploys generated MCP server to Cloud Run, health check passes, SOC2 controls verified |
| D11 | GCP provisioning library (`internal/deploy/gcp/`) | TBD | Provisions all required GCP resources idempotently via Go SDK |
| D12 | Cloud Source Repositories integration | TBD | Pushes generated server code to GCP-hosted git repo |
| D13 | GitHub Actions deploy workflow template | TBD | Workflow triggers on push, deploys to Cloud Run, rolls back on failure |
| D14 | SOC2 security controls documentation | TBD | Documents all security controls, maps to SOC2 trust service criteria |
| D15 | Canary deployment support | TBD | `--canary` flag splits traffic between old and new revisions |

---

## Checkable Work Breakdown

### Epic E13: Deploy Command Foundation

- [x] T13.1 Add `deploy` subcommand dispatch to CLI  Owner: TBD  Est: 45m  Completed: 2026-03-07
  - Add `case "deploy"` to main.go switch statement.
  - Create `cmd/mint/deploy.go` with subcommand dispatch for `gcp`, `status`, `rollback`.
  - Wire up `--help` flag.
  - Acceptance: `mint deploy --help` prints usage. `mint deploy gcp --help` prints GCP-specific flags.
  - Deps: none
  - [x] S13.1.1 Add unit tests for deploy CLI dispatch  Est: 30m
  - [x] S13.1.2 Run linter and formatter  Est: 15m

- [x] T13.2 Define deploy configuration model  Owner: TBD  Est: 1h  Completed: 2026-03-07
  - Create `internal/deploy/config.go` with structs:
    - `DeployConfig` (project ID, region, service name, source dir, image tag, auth settings, canary percentage, vpc, waf, internal, public flags)
    - `DeployResult` (service URL, revision name, status, error)
  - Parse flags into DeployConfig in `cmd/mint/deploy.go`.
  - Flags: `--project`, `--region` (default us-central1), `--service` (default from spec title), `--source` (path to generated server dir), `--public` (default false), `--canary` (percentage, default 0 meaning full rollout), `--vpc`, `--waf`, `--internal`, `--kms-key`, `--timeout` (Cloud Run request timeout, default 300s), `--max-instances` (default 10), `--min-instances` (default 0).
  - Acceptance: All flags parse correctly. Validation rejects missing required flags (project, source).
  - Deps: T13.1
  - [x] S13.2.1 Add unit tests for config parsing and validation  Est: 30m
  - [x] S13.2.2 Run linter and formatter  Est: 15m

- [x] T13.3 Implement GCP authentication helper  Owner: TBD  Est: 45m  Completed: 2026-03-07
  - Create `internal/deploy/gcp/auth.go`.
  - Use `google.golang.org/api/option` and Application Default Credentials.
  - Detect if credentials are available. Print clear error message if not, instructing user to run `gcloud auth application-default login`.
  - Acceptance: Returns authenticated clients. Clear error on missing credentials.
  - Deps: T13.2
  - [x] S13.3.1 Add unit tests for auth helper (mock credentials)  Est: 30m
  - [x] S13.3.2 Run linter and formatter  Est: 15m

### Epic E14: Container Image Build and Registry

- [x] T14.1 Implement Artifact Registry repository provisioning  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/registry.go`.
  - Use `cloud.google.com/go/artifactregistry/apiv1` to create a Docker repository if it does not exist.
  - Repository name: `mint-mcp-servers` in the specified project and region.
  - Enable vulnerability scanning by default.
  - Idempotent: skip creation if repository already exists.
  - Acceptance: Repository created on first run, skipped on subsequent runs. Vulnerability scanning enabled.
  - Deps: T13.3
  - [x] S14.1.1 Add unit tests with mock Artifact Registry client  Est: 30m
  - [x] S14.1.2 Run linter and formatter  Est: 15m

- [x] T14.2 Implement Cloud Build image builder  Owner: TBD  Est: 1.5h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/build.go`.
  - Use `cloud.google.com/go/cloudbuild/apiv1/v2` to submit a build.
  - Upload source directory as a tarball to Cloud Storage (build source bucket).
  - Build steps: multi-stage Dockerfile (already generated by mint). Build produces image tagged with git commit SHA and `latest`.
  - Image pushed to Artifact Registry repository from T14.1.
  - Wait for build completion. Stream build logs to stderr.
  - Acceptance: Build completes in under 60 seconds for petstore server. Image available in Artifact Registry.
  - Deps: T14.1
  - [x] S14.2.1 Add unit tests with mock Cloud Build client  Est: 45m
  - [x] S14.2.2 Run linter and formatter  Est: 15m

- [x] T14.3 Implement distroless Dockerfile update  Owner: TBD  Est: 45m  Completed: 2026-03-07
  - Update the Dockerfile template in `internal/mcpgen/golang/templates/Dockerfile.tmpl` to use `gcr.io/distroless/static-debian12` as the runtime base image.
  - Ensure the binary runs as non-root user (USER nonroot:nonroot).
  - Acceptance: Generated Dockerfile uses distroless base. Container has no shell. Process runs as non-root.
  - Deps: none
  - [x] S14.3.1 Add unit test verifying Dockerfile template output  Est: 30m
  - [x] S14.3.2 Run linter and formatter  Est: 15m

### Epic E15: Cloud Run Deployment with SOC2 Controls

Decision rationale: docs/adr/001-gcp-cloud-run-deployment-target.md and docs/adr/004-soc2-security-controls-for-cloud-run.md.

- [x] T15.1 Implement Cloud Run service provisioning  Owner: TBD  Est: 2h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/cloudrun.go`.
  - Use `cloud.google.com/go/run/apiv2` to create or update a Cloud Run service.
  - Service configuration:
    - Container image from Artifact Registry (from T14.2).
    - Request timeout from `--timeout` flag.
    - Max instances from `--max-instances` flag.
    - Min instances from `--min-instances` flag.
    - CPU allocation: CPU is only allocated during request processing (default). Use `--cpu-always` for SSE transport.
    - Port: 8080 (default Cloud Run port).
    - Startup probe: HTTP GET on `/health` (to be added to generated server).
  - Idempotent: update existing service if it exists, create if not.
  - Acceptance: Cloud Run service created with correct configuration. Service URL returned.
  - Deps: T14.2, T13.2
  - [x] S15.1.1 Add unit tests with mock Cloud Run client  Est: 45m
  - [x] S15.1.2 Run linter and formatter  Est: 15m

- [x] T15.2 Implement IAM policy for Cloud Run service  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - In `internal/deploy/gcp/iam.go`.
  - By default: no `allUsers` or `allAuthenticatedUsers` binding. Service requires IAM identity token to invoke.
  - When `--public` flag is set: add `allUsers` with `roles/run.invoker`. Print warning to stderr.
  - Create a dedicated service account `mint-mcp-<service-name>@<project>.iam.gserviceaccount.com` with only `roles/run.invoker` on itself.
  - Acceptance: Default deployment requires IAM auth. `--public` flag allows unauthenticated access with warning.
  - Deps: T15.1
  - [x] S15.2.1 Add unit tests for IAM policy construction  Est: 30m
  - [x] S15.2.2 Run linter and formatter  Est: 15m

- [x] T15.3 Implement Secret Manager integration  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/secrets.go`.
  - Use `cloud.google.com/go/secretmanager/apiv1`.
  - `--secret` flag accepts `ENV_VAR=secret-name` pairs (repeatable).
  - Create secrets in Secret Manager if they do not exist (user must set values via GCP console or gcloud).
  - Mount secrets as environment variables in Cloud Run service.
  - Grant the service account `roles/secretmanager.secretAccessor` on each secret.
  - Acceptance: Secrets mounted as env vars. Service account has accessor role. Secrets never in container image.
  - Deps: T15.1, T15.2
  - [x] S15.3.1 Add unit tests for Secret Manager provisioning  Est: 30m
  - [x] S15.3.2 Run linter and formatter  Est: 15m

- [x] T15.4 Implement deployment labels and audit metadata  Owner: TBD  Est: 30m  Completed: 2026-03-08
  - Add labels to Cloud Run service:
    - `mint-version`: mint CLI version.
    - `spec-hash`: SHA256 of the source OpenAPI spec (first 12 chars).
    - `commit-sha`: git commit SHA of the source directory (if available).
    - `deployed-by`: username from `os/user.Current()`.
    - `deployed-at`: UTC timestamp.
  - Acceptance: Labels present on deployed Cloud Run service. Queryable via `gcloud run services describe`.
  - Deps: T15.1
  - [x] S15.4.1 Add unit tests for label construction  Est: 15m
  - [x] S15.4.2 Run linter and formatter  Est: 15m

- [x] T15.5 Add health endpoint to generated MCP servers  Owner: TBD  Est: 45m  Completed: 2026-03-07
  - Add `/health` endpoint to the generated server's HTTP handler (in `server.go.tmpl`).
  - Returns 200 OK with `{"status": "ok"}` body.
  - Used by Cloud Run startup probe and by `mint deploy` health check.
  - Acceptance: Generated server responds to `GET /health` with 200. Existing MCP functionality unaffected.
  - Deps: none
  - [x] S15.5.1 Add unit test for health endpoint in generated server  Est: 30m
  - [x] S15.5.2 Run linter and formatter  Est: 15m

- [x] T15.6 Implement post-deploy health check  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - After Cloud Run deployment, send HTTP GET to `<service-url>/health`.
  - Retry up to 5 times with exponential backoff (1s, 2s, 4s, 8s, 16s).
  - If health check passes: print success message with service URL.
  - If health check fails: trigger automatic rollback to previous revision (T15.7).
  - For IAM-authenticated services, use the deployer's credentials to call the health endpoint.
  - Acceptance: Healthy deployment prints URL. Unhealthy deployment triggers rollback.
  - Deps: T15.1, T15.5
  - [x] S15.6.1 Add unit tests for health check with mock HTTP  Est: 30m
  - [x] S15.6.2 Run linter and formatter  Est: 15m

- [x] T15.7 Implement rollback to previous revision  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/rollback.go`.
  - Implement `mint deploy rollback --project P --region R --service S` command.
  - List Cloud Run revisions, shift 100% traffic to the previous revision.
  - Also called automatically from T15.6 on health check failure.
  - Acceptance: Traffic shifts to previous revision. Rollback completes in under 10 seconds.
  - Deps: T15.1
  - [x] S15.7.1 Add unit tests for rollback logic  Est: 30m
  - [x] S15.7.2 Run linter and formatter  Est: 15m

### Epic E16: Cloud Source Repositories Integration

- [x] T16.1 Implement Cloud Source Repositories provisioning  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/sourcerepo.go`.
  - Use `cloud.google.com/go/sourcerepo/apiv1` to create a repository named `mint-mcp-<service-name>`.
  - Idempotent: skip if repository already exists.
  - Acceptance: Repository created in GCP project. URL returned.
  - Deps: T13.3
  - [x] S16.1.1 Add unit tests with mock Source Repository client  Est: 30m
  - [x] S16.1.2 Run linter and formatter  Est: 15m

- [x] T16.2 Implement source code push to Cloud Source Repositories  Owner: TBD  Est: 1.5h  Completed: 2026-03-08
  - After successful deployment, push the generated server source code to the Cloud Source Repository.
  - Use `go-git` library or shell out to `git` to initialize a local repo (if not already), add files, commit, and push.
  - Remote URL: `https://source.developers.google.com/p/<project>/r/mint-mcp-<service-name>`.
  - Use Application Default Credentials for git authentication via credential helper.
  - Acceptance: Source code available in Cloud Source Repository after deploy. Commit message includes spec hash and deploy timestamp.
  - Deps: T16.1, T15.1
  - [x] S16.2.1 Add unit tests for git operations (mock)  Est: 30m
  - [x] S16.2.2 Run linter and formatter  Est: 15m

### Epic E17: Canary Deployments and Traffic Management

- [x] T17.1 Implement canary deployment with traffic splitting  Owner: TBD  Est: 1.5h  Completed: 2026-03-08
  - When `--canary N` flag is provided (N is percentage 1-99), deploy the new revision but route only N% of traffic to it.
  - Remaining traffic stays on the current revision.
  - Print instructions for promoting or rolling back.
  - `mint deploy gcp --promote --project P --region R --service S` shifts 100% traffic to the canary revision.
  - Acceptance: `--canary 10` routes 10% traffic to new revision. `--promote` shifts to 100%.
  - Deps: T15.1
  - [x] S17.1.1 Add unit tests for traffic splitting logic  Est: 30m
  - [x] S17.1.2 Run linter and formatter  Est: 15m

- [x] T17.2 Implement `mint deploy status` command  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - Show current Cloud Run service status: revisions, traffic split, URL, last deploy time, labels.
  - Support `--format json` for machine-readable output.
  - Acceptance: `mint deploy status --project P --region R --service S` prints current state. JSON output is valid.
  - Deps: T15.1
  - [x] S17.2.1 Add unit tests for status output formatting  Est: 30m
  - [x] S17.2.2 Run linter and formatter  Est: 15m

### Epic E18: AI-Native Release Pipeline

Decision rationale: docs/adr/003-ai-native-release-pipeline.md.

- [x] T18.1 Create GitHub Actions deploy workflow template  Owner: TBD  Est: 1.5h  Completed: 2026-03-08
  - Create `templates/workflows/deploy-gcp.yml.tmpl` (embedded template).
  - Workflow triggers on push to main branch when spec file or server source changes.
  - Steps: checkout, setup Go, install mint, generate MCP server (if spec changed), deploy to Cloud Run.
  - Uses Workload Identity Federation for keyless GCP authentication from GitHub Actions.
  - `mint deploy gcp` command generates this workflow when `--ci` flag is passed.
  - Acceptance: Generated workflow file is valid GitHub Actions YAML. Workflow deploys successfully when triggered.
  - Deps: T15.1, T13.2
  - [x] S18.1.1 Add unit test for workflow template rendering  Est: 30m
  - [x] S18.1.2 Run linter and formatter  Est: 15m

- [x] T18.2 Implement Workload Identity Federation setup  Owner: TBD  Est: 1.5h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/workloadidentity.go`.
  - When `--ci` flag is passed, provision:
    - Workload Identity Pool for GitHub Actions.
    - Workload Identity Provider linked to the GitHub repository.
    - Service account with deploy permissions bound to the pool.
  - Print the `workload_identity_provider` and `service_account` values for the GitHub Actions workflow.
  - Idempotent: skip if pool/provider already exist.
  - Acceptance: GitHub Actions workflow can authenticate to GCP without service account keys.
  - Deps: T13.3
  - [x] S18.2.1 Add unit tests with mock IAM/STS clients  Est: 30m
  - [x] S18.2.2 Run linter and formatter  Est: 15m

- [x] T18.3 Implement deploy orchestrator (end-to-end flow)  Owner: TBD  Est: 2h  Completed: 2026-03-08
  - Create `internal/deploy/gcp/deploy.go` as the top-level orchestrator.
  - Orchestration sequence:
    1. Validate DeployConfig.
    2. Authenticate to GCP.
    3. Provision Artifact Registry repository (T14.1).
    4. Build container image via Cloud Build (T14.2).
    5. Provision Cloud Run service (T15.1).
    6. Apply IAM policy (T15.2).
    7. Mount secrets (T15.3, if any).
    8. Apply labels (T15.4).
    9. Run health check (T15.6).
    10. Push source to Cloud Source Repositories (T16.2, if enabled).
    11. Print deploy summary (URL, revision, status).
  - If any step fails, print clear error and exit. If health check fails, rollback.
  - Progress output to stderr. Final URL to stdout (for piping).
  - Acceptance: `mint deploy gcp --project P --region R --source ./server` executes all steps in order. Single command from source to live endpoint.
  - Deps: T14.1, T14.2, T15.1, T15.2, T15.3, T15.4, T15.6, T16.2
  - [x] S18.3.1 Add integration test for full deploy flow (mock all GCP clients)  Est: 1h
  - [x] S18.3.2 Run linter and formatter  Est: 15m

### Epic E19: Testing and Documentation

- [x] T19.1 Add integration tests for deploy command  Owner: TBD  Est: 2h  Completed: 2026-03-08
  - Test the full deploy flow with mocked GCP SDK clients.
  - Test error paths: missing credentials, build failure, health check failure, rollback.
  - Test idempotency: deploy twice, verify no errors.
  - Test canary: deploy with `--canary 10`, verify traffic split.
  - Acceptance: All integration tests pass. 80%+ coverage on `internal/deploy/` package.
  - Deps: T18.3
  - [x] S19.1.1 Run linter and formatter  Est: 15m

- [x] T19.2 Add deploy command to CLI help text  Owner: TBD  Est: 30m  Completed: 2026-03-08
  - Update `printUsage()` in main.go to include `deploy` command.
  - Add `--help` text for `deploy gcp`, `deploy status`, `deploy rollback`.
  - Acceptance: `mint help` lists deploy. `mint deploy --help` lists subcommands. All flags documented.
  - Deps: T13.1
  - [x] S19.2.1 Run linter and formatter  Est: 15m

- [x] T19.3 Write deploy documentation in README  Owner: TBD  Est: 1h  Completed: 2026-03-08
  - Add deploy section to README covering:
    - Prerequisites (GCP project, billing, gcloud CLI for auth).
    - Quickstart: generate MCP server, deploy to Cloud Run.
    - Security controls and SOC2 compliance.
    - Canary deployments.
    - CI/CD setup with GitHub Actions.
    - Rollback procedure.
  - Acceptance: README section is complete with examples. No missing steps.
  - Deps: T18.3
  - [x] S19.3.1 Run linter and formatter  Est: 15m

- [x] T19.4 Run final linter and formatter pass on all deploy code  Owner: TBD  Est: 30m  Completed: 2026-03-08
  - `golangci-lint run ./internal/deploy/...`
  - `golangci-lint run ./cmd/mint/...`
  - `gofmt -s -w .`
  - Acceptance: Zero lint findings. Zero formatting changes.
  - Deps: all E13-E18 tasks

### Archived

- **E1 through E12 (v0.1.0)** -- Completed. All tasks trimmed. Stable knowledge preserved in docs/design.md.

---

## Parallel Work

### Track A: CLI and Config (E13)

Tasks: T13.1, T13.2, T13.3

### Track B: Container Build (E14)

Tasks: T14.1, T14.2, T14.3

### Track C: Cloud Run + Security (E15)

Tasks: T15.1, T15.2, T15.3, T15.4, T15.5, T15.6, T15.7

### Track D: Source Repository (E16)

Tasks: T16.1, T16.2

### Parallel Execution

| Phase | Track A | Track B | Track C | Track D |
|-------|---------|---------|---------|---------|
| Phase 1 | T13.1, T13.2 | T14.3 | T15.5 | -- |
| Phase 2 | T13.3 | -- | -- | -- |
| Phase 3 | -- | T14.1 | -- | T16.1 |
| Phase 4 | -- | T14.2 | T15.1 | T16.2 |
| Phase 5 | -- | -- | T15.2, T15.3, T15.4 | -- |
| Phase 6 | -- | -- | T15.6, T15.7 | -- |

**Sync points:**
- T13.3 (auth) must complete before T14.1, T15.1, T16.1 can start.
- T14.2 (build) must complete before T15.1 (Cloud Run deploy) can start.
- T15.1 must complete before T15.2, T15.3, T15.4, T15.6, T15.7.
- T14.3 and T15.5 have no dependencies and can run in Phase 1 alongside Track A.
- E17 (canary) and E18 (pipeline) depend on E15 completion.
- E19 (testing/docs) runs last after all implementation is complete.

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M6: Deploy Foundation | E13, E14 | None | `mint deploy gcp --help` works. Container image builds via Cloud Build. Distroless Dockerfile template updated. |
| M7: Cloud Run Live | E15 | M6 | MCP server deployed to Cloud Run with SOC2 controls. Health check passes. Rollback works. |
| M8: Source + Canary | E16, E17 | M7 | Source code in Cloud Source Repos. Canary deployments with traffic splitting. Status command works. |
| M9: AI-Native Pipeline | E18 | M7 | GitHub Actions workflow deploys on push. Workload Identity Federation configured. End-to-end orchestrator tested. |
| M10: Ship Deploy | E19 | M7, M8, M9 | Integration tests pass. README updated. CLI help complete. v0.2.0 released. |

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R9 | GCP Go SDK API breaking changes | Medium | Low | Pin SDK versions in go.mod. Wrap SDK calls in internal adapters. |
| R10 | Cloud Build quotas throttle high-frequency deploys | Medium | Medium | Use regional Cloud Build. Document quota increase request process. Consider pre-built images for unchanged code. |
| R11 | SSE connections dropped by Cloud Run default timeout | High | High | Set request timeout to 3600s for SSE transport. Document `--cpu-always` flag for persistent connections. |
| R12 | Users lack GCP project setup knowledge | Medium | High | Provide clear prerequisites in README. Print actionable error messages for common setup issues (billing not enabled, APIs not enabled). |
| R13 | Workload Identity Federation setup complexity | Medium | Medium | Automate setup via `--ci` flag. Print step-by-step instructions if manual setup is needed. |
| R14 | Distroless containers limit debugging | Low | Medium | Provide `--debug-image` flag for non-production deployments that uses alpine base with shell. |
| R15 | Cloud Source Repositories being deprecated | Medium | Low | Make source repo push optional (enabled by default, disabled with `--no-source-repo`). Source code is always available locally. |

---

## Operating Procedure

### Definition of Done

A task is done when:
1. Code compiles with zero warnings.
2. All new code has unit tests that pass.
3. Integration tests pass for the affected command.
4. `golangci-lint run` passes with no new findings.
5. `gofmt -s` produces no changes.
6. CLI help text is accurate for any new/changed commands.
7. GCP SDK calls are wrapped with clear error messages for common failure modes.
8. Security controls are verified (IAM, secrets, non-root, TLS).

### Review and QA Steps

1. Self-review all changed files before marking a task complete.
2. Run `go test ./...` and verify no regressions.
3. Run `golangci-lint run` and fix any findings.
4. Run `gofmt -s -w .` to ensure formatting.
5. For deploy tasks: verify with a real GCP project that the deployment succeeds (or verify mocks cover all API calls).
6. For security tasks: verify the deployed service configuration matches SOC2 requirements.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Never allow changes to pile up. Make many small logical commits.
- Each commit should represent one logical change and have a clear message.

---

## Progress Log

### 2026-03-07 -- Change Summary (Plan Created)

- Trimmed completed epics E1 through E12 (v0.1.0). Stable knowledge preserved in docs/design.md.
- Created new plan for `mint deploy` command targeting Google Cloud Platform.
- Added 7 new epics: E13 (Deploy Foundation), E14 (Container Build), E15 (Cloud Run + SOC2), E16 (Source Repos), E17 (Canary), E18 (AI Pipeline), E19 (Testing/Docs).
- Total: 22 tasks, 44 subtasks.
- Created 4 ADRs:
  - docs/adr/001-gcp-cloud-run-deployment-target.md -- Cloud Run as initial deploy target.
  - docs/adr/002-go-sdk-over-terraform-for-provisioning.md -- Go SDK over Terraform.
  - docs/adr/003-ai-native-release-pipeline.md -- AI-native release pipeline design.
  - docs/adr/004-soc2-security-controls-for-cloud-run.md -- SOC2 security controls.
- 5 milestones defined: M6 through M10.
- 7 risks identified: R9 through R15.

### 2026-03-07 -- Plan Created

- No implementation progress yet. Plan is new.

---

## Hand-off Notes

### What You Need to Know

1. **Context**: Mint is a Go CLI that generates MCP servers from OpenAPI specs. All generation features (E1-E12) are complete. This plan adds cloud deployment capabilities starting with GCP Cloud Run.
2. **Previous work**: See docs/design.md for architecture, conventions, and completed milestones.
3. **New dependencies**: GCP Go SDK packages (`cloud.google.com/go/run`, `cloud.google.com/go/artifactregistry`, `cloud.google.com/go/cloudbuild`, `cloud.google.com/go/secretmanager`, `cloud.google.com/go/sourcerepo`, `cloud.google.com/go/iam`).
4. **Security**: SOC2 controls are enforced by default. Permissive options (`--public`, `--debug-image`) require explicit flags.
5. **Testing**: Mock all GCP SDK clients for unit and integration tests. Real GCP project needed only for manual validation.
6. **Key files**:
   - `cmd/mint/deploy.go` -- CLI command and flag parsing.
   - `internal/deploy/config.go` -- Deploy configuration model.
   - `internal/deploy/gcp/` -- All GCP-specific provisioning and deployment logic.
   - `internal/deploy/gcp/deploy.go` -- Top-level orchestrator.

### Credentials and Links (Placeholders)

- GCP project for testing: TBD (create a dedicated test project).
- GitHub org: `github.com/sirerun` -- existing.
- Workload Identity Federation pool: created by `mint deploy gcp --ci`.
- No API keys or secrets stored in repository.

---

## Appendix

### Deploy Command Examples

**Basic deploy:**
```
mint mcp generate petstore.yaml --output ./server
mint deploy gcp --project my-project --region us-central1 --source ./server
```

**Deploy with secrets:**
```
mint deploy gcp --project my-project --source ./server \
  --secret API_KEY=petstore-api-key \
  --secret DB_PASSWORD=petstore-db-pass
```

**Canary deploy:**
```
mint deploy gcp --project my-project --source ./server --canary 10
# After validation:
mint deploy gcp --promote --project my-project --service petstore-mcp
```

**Check status:**
```
mint deploy status --project my-project --service petstore-mcp
mint deploy status --project my-project --service petstore-mcp --format json
```

**Rollback:**
```
mint deploy rollback --project my-project --service petstore-mcp
```

**Setup CI/CD:**
```
mint deploy gcp --project my-project --source ./server --ci
# Generates .github/workflows/deploy-gcp.yml and provisions Workload Identity Federation
```

### GCP APIs Required

The following GCP APIs must be enabled in the target project. `mint deploy gcp` will check and print instructions if any are missing:

- Cloud Run Admin API (`run.googleapis.com`)
- Cloud Build API (`cloudbuild.googleapis.com`)
- Artifact Registry API (`artifactregistry.googleapis.com`)
- Secret Manager API (`secretmanager.googleapis.com`)
- Source Repo API (`sourcerepo.googleapis.com`)
- IAM API (`iam.googleapis.com`)
- Cloud Resource Manager API (`cloudresourcemanager.googleapis.com`)

### SOC2 Control Mapping

| SOC2 Criteria | Control | Implementation |
|---------------|---------|----------------|
| CC6.1 Logical access | IAM-based authentication | Cloud Run `--no-allow-unauthenticated` default |
| CC6.1 Least privilege | Dedicated service account | Per-deployment SA with minimal roles |
| CC6.6 Encryption in transit | TLS 1.2+ | Cloud Run built-in TLS termination |
| CC6.7 Encryption at rest | Google-managed encryption | Artifact Registry and Secret Manager defaults |
| CC7.1 Monitoring | Cloud Audit Logs | Enabled by default for Cloud Run admin operations |
| CC7.2 Change management | Deployment labels | Commit SHA, spec hash, deployer, timestamp on every revision |
| CC8.1 Change control | Immutable containers | Distroless base, no shell, non-root execution |
