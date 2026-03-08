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

### Key Dependencies

| Dependency | Purpose | Used In |
|-----------|---------|---------|
| pb33f/libopenapi | OpenAPI parsing | mint binary |
| mark3labs/mcp-go | Go MCP SDK | Generated servers only |

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

## Key File Paths

| Path | Description |
|------|-------------|
| cmd/mint/main.go | CLI entry point and subcommand dispatch |
| cmd/mint/mcp.go | `mint mcp generate` command |
| internal/mcpgen/model.go | MCP model structs (MCPServer, MCPTool, MCPToolParam, MCPAuth) |
| internal/mcpgen/converter.go | OpenAPI-to-MCP model converter |
| internal/mcpgen/golang/generate.go | Go code generation orchestrator |
| internal/mcpgen/golang/templates/ | Embedded Go templates (7 files) |
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

## References

- MCP Specification: https://modelcontextprotocol.io
- OpenAPI Specification: https://spec.openapis.org/oas/v3.1.0
- pb33f/libopenapi: https://github.com/pb33f/libopenapi
- mark3labs/mcp-go: https://github.com/mark3labs/mcp-go
