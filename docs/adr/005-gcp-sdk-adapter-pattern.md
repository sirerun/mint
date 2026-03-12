# ADR 005: GCP SDK Adapter Pattern for Deploy

## Status
Accepted

## Date
2026-03-11

## Context
The deploy feature defines 8 interfaces in `internal/deploy/gcp/` (RegistryClient, BuildClient, CloudRunClient, IAMPolicyClient, SecretClient, SourceRepoClient, GitClient, TrafficClient/RevisionClient/StatusClient). Business logic and orchestration are complete and tested against mocks. The missing piece is concrete adapter implementations that connect these interfaces to real GCP SDK clients.

We need to decide how to structure these adapters: inline in existing files, in a separate sub-package, or as standalone adapter files in the same package.

## Decision
Create one adapter file per GCP service in `internal/deploy/gcp/` following the naming convention `<service>_adapter.go` (e.g., `cloudrun_adapter.go`, `registry_adapter.go`). Each adapter file contains a struct that wraps the real GCP SDK client and implements the corresponding interface. The adapter constructor accepts `context.Context` and project/region config, creates the SDK client, and returns the adapter.

The CLI entry point (`cmd/mint/deploy.go`) instantiates all adapters and injects them into the Deployer orchestrator. For the git operations adapter, use `os/exec` to shell out to the `git` binary rather than adding a Go git library dependency.

## Consequences
- Positive: Clean separation between business logic (tested with mocks) and SDK glue code. Each adapter is small and focused.
- Positive: Existing tests continue to work unchanged since they test against interfaces.
- Positive: Adding adapters for other cloud providers in the future follows the same pattern.
- Negative: Adapter code is harder to unit test (requires real GCP credentials or complex SDK mocking). Mitigated by integration tests.
- Negative: Multiple new GCP SDK dependencies added to go.mod.
