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
  deploy/           Deploy configuration model and validation
  deploy/gcp/       GCP deployment orchestration (interfaces, business logic, adapters)
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

The deploy feature uses an interface-based dependency injection architecture. All layers are fully implemented:

- `internal/deploy/config.go` -- DeployConfig struct, validation, flag parsing.
- `internal/deploy/gcp/deploy.go` -- Deployer orchestrator with 8 pluggable interfaces.
- `internal/deploy/gcp/*.go` -- Interface definitions and business logic for each concern (registry, build, cloudrun, iam, secrets, sourcerepo, sourcepush, status, rollback, canary, healthcheck, workloadidentity, labels, workflow).
- `internal/deploy/gcp/*_adapter.go` -- Concrete GCP SDK adapter implementations for all interfaces.
- `internal/deploy/gcp/adapters.go` -- Bridge adapter layer connecting low-level SDK client interfaces to high-level Deployer orchestrator interfaces.
- `internal/deploy/gcp/apis.go` -- GCP API enablement check (verifies required APIs are enabled before deploy).
- `cmd/mint/deploy.go` -- CLI flag parsing, adapter instantiation, and orchestration calls for `gcp`, `status`, and `rollback` subcommands.

**Adapter architecture:** CloudRunAdapter is split into 4 sub-adapter structs (CloudRunServiceAdapter, CloudRunStatusAdapter, CloudRunRevisionAdapter, CloudRunTrafficAdapter) because Go does not allow methods with the same name but different return types on one struct. See docs/adr/005-gcp-sdk-adapter-pattern.md.

### Key Dependencies

| Dependency | Purpose | Used In |
|-----------|---------|---------|
| pb33f/libopenapi | OpenAPI parsing | mint binary |
| mark3labs/mcp-go | Go MCP SDK | Generated servers only |
| cloud.google.com/go/artifactregistry | Artifact Registry SDK | mint binary (adapter) |
| cloud.google.com/go/run/apiv2 | Cloud Run Admin API v2 | mint binary (adapter) |
| cloud.google.com/go/cloudbuild/apiv1/v2 | Cloud Build API | mint binary (adapter) |
| cloud.google.com/go/secretmanager/apiv1 | Secret Manager API | mint binary (adapter) |
| cloud.google.com/go/iam/admin/apiv1 | IAM Admin API | mint binary (adapter) |
| cloud.google.com/go/storage | Cloud Storage (source upload) | mint binary (build adapter) |
| google.golang.org/api/serviceusage/v1 | Service Usage API | mint binary (API check) |
| google.golang.org/api/sourcerepo/v1 | Source Repos REST API (deprecated) | mint binary (adapter) |

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

### Deploy IAM Roles

Required IAM roles for the deployer service account:
- `roles/run.admin`
- `roles/artifactregistry.admin`
- `roles/cloudbuild.builds.editor`
- `roles/secretmanager.admin`
- `roles/iam.serviceAccountAdmin`

## Key File Paths

| Path | Description |
|------|-------------|
| cmd/mint/main.go | CLI entry point and subcommand dispatch |
| cmd/mint/mcp.go | `mint mcp generate` command |
| cmd/mint/deploy.go | Deploy CLI dispatch (gcp, status, rollback) -- fully wired |
| internal/deploy/gcp/adapters.go | Bridge adapters (8 types) connecting SDK clients to Deployer |
| internal/deploy/gcp/*_adapter.go | GCP SDK adapter implementations (registry, build, cloudrun, iam, secrets, sourcerepo, git) |
| internal/deploy/gcp/apis.go | GCP API enablement check |
| internal/mcpgen/model.go | MCP model structs (MCPServer, MCPTool, MCPToolParam, MCPAuth) |
| internal/mcpgen/converter.go | OpenAPI-to-MCP model converter |
| internal/mcpgen/golang/generate.go | Go code generation orchestrator |
| internal/mcpgen/golang/templates/ | Embedded Go templates (7 files) |
| internal/deploy/config.go | DeployConfig, validation, SecretMapping |
| internal/deploy/gcp/deploy.go | Deployer orchestrator (8 interface deps) |
| .goreleaser.yml | Cross-platform release config |
| action.yml | GitHub Action for lint/validate/diff in CI |

## Completed Milestones

| Milestone | Date | Description |
|-----------|------|-------------|
| M1: Foundation | 2026-03 | CLI compiles, loads specs, CI green |
| M2: Core OpenAPI Tools | 2026-03 | Lint, diff, merge, overlay, transform commands work |
| M3: MCP Generation | 2026-03 | `mint mcp generate` produces working Go MCP servers |
| M4: MCP Advanced + CI/CD | 2026-03 | Auth, SSE, filtering, GitHub Actions |
| M5: Ship It | 2026-03 | README, examples, v0.1.0 release |
| M6-M10: Deploy Scaffold | 2026-03 | Interface design, business logic, mock tests for deploy feature |
| M11: Adapters Complete | 2026-03 | All 8 GCP SDK adapter files compile, interface checks pass |
| M12: CLI Wired | 2026-03 | deploy gcp, status, rollback execute real GCP calls |
| M13: Production Ready | 2026-03 | Manual e2e validation passes with Twitter API v2 spec |

## References

- MCP Specification: https://modelcontextprotocol.io
- OpenAPI Specification: https://spec.openapis.org/oas/v3.1.0
- pb33f/libopenapi: https://github.com/pb33f/libopenapi
- mark3labs/mcp-go: https://github.com/mark3labs/mcp-go
