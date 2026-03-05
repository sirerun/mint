# Mint -- Open Source OpenAPI-to-MCP Toolchain

## Project Name Candidates

The project lives under the **Sire Run** organization (`github.com/sirerun`). Five name candidates that align with the brand identity -- evoking authority ("sire"), motion ("run"), and craftsmanship:

1. **Sireforge** -- The sire forges MCP servers from API specs. Strong, commanding, implies craftsmanship. `github.com/sirerun/sireforge`
2. **Rungate** -- A gateway that runs your API as MCP tools. Compact, memorable, plays on "run". `github.com/sirerun/rungate`
3. **Siremcp** -- Direct and unambiguous. The sire of MCP servers. Easy to search, easy to type. `github.com/sirerun/siremcp`
4. **Ironspec** -- Iron-clad specs, forged into servers. Pairs well with "sirerun" as the org. `github.com/sirerun/ironspec`
5. **Sirecast** -- The sire casts (generates) servers from molds (specs). Evokes metalwork and precision. `github.com/sirerun/sirecast`

The plan below uses **Mint** as a placeholder. Replace with the chosen name before implementation begins.

---

## Context

### Problem Statement

The Model Context Protocol (MCP) is rapidly becoming the standard way AI agents interact with external services. Today, building an MCP server from an existing API requires manually translating an OpenAPI spec into tool definitions, writing transport handlers, and wiring up authentication -- tedious, error-prone work that developers repeat for every API.

The Speakeasy CLI offers MCP server generation but locks users into a proprietary platform with mandatory authentication, telemetry, and closed-source generation engines. There is no open-source tool that takes an OpenAPI spec and produces a fully functional, ready-to-deploy MCP server.

**Mint** is an open-source Go CLI that turns any OpenAPI 3.0/3.1 specification into a working Go MCP server. It also provides the foundational OpenAPI tooling (validation, linting, diffing, merging, transformation) needed to prepare specs for generation. The ultimate deliverable is: `mint mcp generate spec.yaml --output ./server` produces a deployable MCP server written in Go.

### Objectives

1. **Primary goal**: Generate production-quality Go MCP servers from OpenAPI specs with a single command.
2. Build foundational OpenAPI tooling (lint, diff, merge, transform, overlay) as prerequisites for clean MCP generation.
3. Produce MCP servers that conform to the MCP specification with stdio and HTTP/SSE transports.
4. Map OpenAPI operations to MCP tools with typed input schemas derived from request parameters and bodies.
5. Support authentication passthrough (API key, Bearer token, OAuth2).
6. Release under Apache 2.0 license at `github.com/sirerun/<chosen-name>`.
7. Deliver a polished developer experience: clear errors, JSON output, offline operation, no telemetry.

### Non-Goals

- Full-parity SDK generation with Speakeasy (10+ languages).
- Platform/SaaS features (billing, registry, studio, accounts).
- Terraform provider generation.
- Agent/AI skill integrations.
- Visual/web-based UI.
- MCP client generation (only servers).
- TypeScript MCP server generation (Go only for v1).

### Constraints and Assumptions

- Written in Go, using the standard library and minimal dependencies.
- CLI built with the standard `flag` package, not cobra/viper.
- Uses `pb33f/libopenapi` (BSD-3) for OpenAPI parsing.
- Uses `pb33f/vacuum` (MIT) for linting engine.
- No dependency on any Speakeasy proprietary libraries.
- Must compile to a single static binary for Linux, macOS, and Windows.
- All output must be deterministic for CI/CD use.
- Generated MCP servers use only the Go standard library and `github.com/mark3labs/mcp-go` SDK.
- MCP specification version targeted: latest stable (2025-03-26 or later).

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| MCP server generation from petstore spec | Produces a working server with all operations as tools | Manual test: connect Claude Desktop to generated server |
| Spec validation accuracy | Catch 95%+ of common OpenAPI errors | Comparison against vacuum/spectral test suites |
| CLI response time | Under 2s for specs up to 50K lines | Benchmarking with real-world specs |
| Test coverage | 80%+ line coverage | `go test -cover` |
| GitHub stars (6 months) | 500+ | GitHub insights |
| Generated MCP servers used in production | 5+ public repos using mint-generated servers | GitHub search |

---

## Scope and Deliverables

### In Scope

1. **OpenAPI Parsing** -- Load, parse, and resolve OpenAPI 3.0/3.1 specs from files, URLs, and stdin.
2. **Validation and Linting** -- Validate specs with built-in and custom rulesets. Surface errors, warnings, and info with file/line references.
3. **Spec Diffing** -- Compare two OpenAPI specs and output breaking/non-breaking changes.
4. **Spec Merging** -- Merge multiple OpenAPI documents into one, with conflict detection.
5. **Overlay Application** -- Apply OpenAPI Overlay documents per the Overlay specification.
6. **Spec Transformation** -- Filter operations, remove unused components, normalize, format, convert Swagger 2.0.
7. **MCP Server Generation (Go)** -- Generate a complete, deployable Go MCP server from an OpenAPI spec. This is the primary deliverable.
8. **CI/CD Integration** -- GitHub Action for running mint in pipelines.
9. **Machine-Readable Output** -- JSON output mode for all commands.

### Out of Scope

- Full production-grade SDK generation (client libraries in 10+ languages).
- TypeScript or other non-Go MCP server generation (may be added in a future version).
- SaaS platform features (auth, billing, registry).
- Terraform provider generation.
- MCP client generation.
- Visual/web-based UI.
- OpenAPI 2.0 (Swagger) native support (conversion from 2.0 is in scope).
- Code sample generation as a standalone feature (samples are embedded in generated MCP servers).

### Deliverables Table

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D1 | CLI binary (`mint`) with all commands | TBD | All commands pass integration tests, binary runs on Linux/macOS/Windows |
| D2 | Linting engine with 3 built-in rulesets | TBD | Rulesets produce correct results on 20+ test specs |
| D3 | Spec diff engine | TBD | Detects breaking and non-breaking changes, JSON output matches expected |
| D4 | Spec merge engine | TBD | Merges 3+ specs with conflict detection, output is valid OpenAPI |
| D5 | Overlay engine | TBD | Applies overlay per specification, output matches expected |
| D6 | Go MCP server generator | TBD | Generated server starts, registers tools for all operations, handles requests via stdio and HTTP/SSE |
| D7 | GitHub Action | TBD | Action runs in workflow, validates spec, posts results to PR |
| D8 | Documentation (README, CLI help, examples) | TBD | README covers install, quickstart, all commands, MCP server usage guide |
| D9 | Release pipeline (goreleaser) | TBD | Tags produce GitHub releases with binaries for 3 platforms |

---

## Checkable Work Breakdown

### Epic E1: Project Bootstrap

- [x] T1.1 Initialize Go module at `github.com/sirerun/mint`  Owner: TBD  Est: 30m
  - Acceptance: `go build ./...` succeeds. Module path is correct.
- [x] T1.2 Set up directory structure (`cmd/`, `internal/`, `pkg/`, `testdata/`, `templates/`)  Owner: TBD  Est: 30m
  - Acceptance: Directories exist. A placeholder `main.go` compiles. `templates/` has subdirectory `mcp-go/`.
- [x] T1.3 Create CLI entry point with `flag` package and subcommand dispatch  Owner: TBD  Est: 1h
  - Acceptance: `mint help`, `mint version`, and unknown subcommands produce correct output.
  - Risk: Subcommand dispatch with `flag` requires manual routing. Keep it simple with a switch statement.
- [x] T1.4 Add unit tests for CLI dispatch  Owner: TBD  Est: 30m
  - Acceptance: Tests cover help, version, unknown command.
- [x] T1.5 Set up CI with GitHub Actions (build, test, lint)  Owner: TBD  Est: 1h
  - Acceptance: Push triggers build+test+lint. Badge in README.
  - Deps: T1.1, T1.2
- [x] T1.6 Configure golangci-lint with a baseline config  Owner: TBD  Est: 30m
  - Acceptance: `golangci-lint run` passes on initial codebase.
  - Deps: T1.2
- [x] T1.7 Set up goreleaser for cross-platform binary releases  Owner: TBD  Est: 1h
  - Acceptance: `goreleaser release --snapshot` produces binaries for linux/darwin/windows amd64+arm64.
  - Deps: T1.1, T1.3

### Epic E2: OpenAPI Parsing and Loading

- [x] T2.1 Implement spec loader (file path, URL, stdin)  Owner: TBD  Est: 1.5h
  - Acceptance: Loads YAML and JSON specs from local files, HTTP URLs, and stdin pipe.
  - Uses `pb33f/libopenapi` for parsing.
- [x] T2.2 Implement spec resolution (resolve `$ref` references)  Owner: TBD  Est: 1h
  - Acceptance: Specs with local and remote `$ref` references are fully resolved.
  - Note: libopenapi handles $ref resolution via AllowFileReferences/AllowRemoteReferences config.
  - Deps: T2.1
- [x] T2.3 Add error reporting with file path and line number  Owner: TBD  Est: 1h
  - Acceptance: Parse errors include file name and line number. JSON output mode includes structured error objects.
  - Deps: T2.1
- [x] T2.4 Add unit and integration tests for spec loading  Owner: TBD  Est: 1h
  - Acceptance: Tests cover YAML, JSON, URL loading, invalid input, stdin. 90%+ coverage for loader package.
  - Deps: T2.1, T2.2, T2.3
- [x] T2.5 Run linter and formatter on E2 code  Owner: TBD  Est: 15m
  - Deps: T2.1, T2.2, T2.3

### Epic E3: Validation and Linting

- [x] T3.1 Integrate `pb33f/vacuum` as linting backend  Owner: TBD  Est: 1.5h
  - Note: vacuum repo not available. Implemented custom validation using libopenapi directly.
  - Deps: T2.1
- [ ] T3.2 Implement `mint lint` command  Owner: TBD  Est: 1h
  - Acceptance: `mint lint spec.yaml` outputs errors/warnings with severity, rule ID, path, line number.
  - Deps: T3.1, T1.3
- [x] T3.3 Add JSON output mode for lint results  Owner: TBD  Est: 45m
  - Acceptance: `mint validate --format json spec.yaml` outputs valid JSON array of diagnostics.
  - Deps: T3.2
- [ ] T3.4 Implement configurable rulesets (recommended, strict, minimal)  Owner: TBD  Est: 1.5h
  - Acceptance: `--ruleset` flag selects ruleset. Custom ruleset file path accepted.
  - Deps: T3.1
- [x] T3.5 Implement `mint validate` command (structural validation only)  Owner: TBD  Est: 1h
  - Acceptance: Reports structural OpenAPI compliance errors (missing required fields, invalid types).
  - Deps: T2.1, T1.3
- [ ] T3.6 Add colored terminal output for lint/validate results  Owner: TBD  Est: 45m
  - Acceptance: Errors in red, warnings in yellow, info in blue. Colors disabled when not a TTY.
  - Deps: T3.2, T3.5
- [x] T3.7 Add unit and integration tests for linting  Owner: TBD  Est: 1h
  - Acceptance: Tests cover JSON output, known-bad specs produce expected diagnostics.
  - Deps: T3.2, T3.3, T3.4, T3.5
- [x] T3.8 Run linter and formatter on E3 code  Owner: TBD  Est: 15m
  - Deps: T3.1 through T3.6

### Epic E4: Spec Diffing

- [x] T4.1 Implement diff engine using `pb33f/openapi-changes`  Owner: TBD  Est: 2h
  - Note: Implemented custom diff engine (openapi-changes repo unavailable). Compares paths, operations, parameters.
  - Deps: T2.1
- [x] T4.2 Implement `mint diff` command  Owner: TBD  Est: 1h
  - Acceptance: `mint diff old.yaml new.yaml` outputs human-readable change list. `--format json` outputs structured JSON.
  - Deps: T4.1, T1.3
- [x] T4.3 Add breaking change detection and exit code  Owner: TBD  Est: 45m
  - Acceptance: Exit code 1 when breaking changes found, 0 otherwise. `--fail-on-breaking` flag.
  - Deps: T4.2
- [x] T4.4 Add unit and integration tests for diffing  Owner: TBD  Est: 1h
  - Acceptance: Tests cover additions, removals, modifications, breaking vs non-breaking. JSON output validated.
  - Deps: T4.1, T4.2, T4.3
- [x] T4.5 Run linter and formatter on E4 code  Owner: TBD  Est: 15m
  - Deps: T4.1, T4.2, T4.3

### Epic E5: Spec Merging

- [x] T5.1 Implement merge engine for combining multiple OpenAPI documents  Owner: TBD  Est: 2h
  - Acceptance: Merges paths, components, tags from 2+ specs. Detects and reports conflicts (duplicate paths, operationIds).
  - Deps: T2.1
- [x] T5.2 Implement `mint merge` command  Owner: TBD  Est: 1h
  - Acceptance: `mint merge a.yaml b.yaml -o merged.yaml` produces valid merged spec. Conflicts reported to stderr.
  - Deps: T5.1, T1.3
- [x] T5.3 Add conflict resolution strategies (fail, rename, skip)  Owner: TBD  Est: 1h
  - Acceptance: `--on-conflict` flag controls behavior. Default is fail.
  - Deps: T5.2
- [x] T5.4 Add unit and integration tests for merging  Owner: TBD  Est: 1.5h
  - Acceptance: Tests cover 2-spec merge, 3-spec merge, conflicts, resolution strategies.
  - Deps: T5.1, T5.2, T5.3
- [x] T5.5 Run linter and formatter on E5 code  Owner: TBD  Est: 15m
  - Deps: T5.1, T5.2, T5.3

### Epic E6: Overlay Application

- [x] T6.1 Implement OpenAPI Overlay specification parser  Owner: TBD  Est: 1.5h
  - Acceptance: Parses overlay YAML/JSON documents per the Overlay specification.
  - Deps: T2.1
- [x] T6.2 Implement overlay application engine  Owner: TBD  Est: 2h
  - Acceptance: Applies actions (update, remove) to target spec via JSONPath selectors. Output is valid OpenAPI.
  - Deps: T6.1
- [x] T6.3 Implement `mint overlay` command  Owner: TBD  Est: 45m
  - Acceptance: `mint overlay apply spec.yaml overlay.yaml -o out.yaml` produces correct output.
  - Deps: T6.2, T1.3
- [x] T6.4 Add unit and integration tests for overlay  Owner: TBD  Est: 1h
  - Acceptance: Tests cover update actions, remove actions, nested paths, invalid overlay documents.
  - Deps: T6.1, T6.2, T6.3
- [x] T6.5 Run linter and formatter on E6 code  Owner: TBD  Est: 15m
  - Deps: T6.1, T6.2, T6.3

### Epic E7: Spec Transformation

- [x] T7.1 Implement filter-operations transform (by tag, path pattern, method)  Owner: TBD  Est: 1.5h
  - Acceptance: Filters operations and removes unused components after filtering.
  - Deps: T2.1
- [x] T7.2 Implement remove-unused-components transform  Owner: TBD  Est: 1h
  - Acceptance: Removes schemas, parameters, responses not referenced by any operation.
  - Deps: T2.1
- [x] T7.3 Implement format/normalize transform (sort keys, consistent style)  Owner: TBD  Est: 1h
  - Acceptance: Output has sorted keys, consistent indentation. Idempotent (running twice produces same output).
  - Deps: T2.1
- [ ] T7.4 Implement Swagger 2.0 to OpenAPI 3.0 conversion  Owner: TBD  Est: 2h
  - Acceptance: Converts Swagger 2.0 petstore to valid OpenAPI 3.0. Handles definitions, parameters, responses.
  - Deps: T2.1
- [x] T7.5 Implement `mint transform` command with subcommands  Owner: TBD  Est: 1h
  - Acceptance: `mint transform filter`, `mint transform cleanup`, `mint transform format` all work.
  - Note: Swagger 2.0 convert deferred to future release.
  - Deps: T7.1, T7.2, T7.3, T1.3
- [x] T7.6 Add unit and integration tests for transformations  Owner: TBD  Est: 1.5h
  - Acceptance: Each transform tested with before/after specs. Idempotency verified for format.
  - Deps: T7.1 through T7.5
- [x] T7.7 Run linter and formatter on E7 code  Owner: TBD  Est: 15m
  - Deps: T7.1 through T7.5

### Epic E8: MCP Server Generation -- Core Engine

This is the primary epic. It builds the engine that maps OpenAPI operations to MCP tools and generates server code.

- [x] T8.1 Design OpenAPI-to-MCP mapping model  Owner: TBD  Est: 1.5h
  - Acceptance: Go struct definitions in `internal/mcpgen/model.go` that represent:
    - `MCPServer` (name, version, description, tools, auth config)
    - `MCPTool` (name from operationId, description from summary/description, inputSchema from parameters+requestBody, HTTP method, path, response schema)
    - `MCPToolParam` (name, type, description, required, enum values, default)
    - `MCPAuth` (type: apiKey/bearer/oauth2, header name, env var name)
  - Document: each OpenAPI operation becomes one MCP tool. OperationId becomes the tool name (converted to snake_case). Path params, query params, header params, and request body fields become tool input properties. The tool's inputSchema is a JSON Schema object derived from the OpenAPI parameter schemas and request body schema.
  - Deps: T2.1

- [x] T8.2 Implement OpenAPI-to-MCP model converter  Owner: TBD  Est: 2h
  - Acceptance: Given a parsed OpenAPI document, produces a list of `MCPTool` structs with:
    - Tool name derived from operationId (snake_case). If no operationId, derive from method+path (e.g., `get_users_by_id`).
    - Tool description from operation summary (fallback to description, fallback to "No description").
    - Input schema combining path params, query params, and request body properties into a single flat JSON Schema object.
    - Required array listing all required path params and required body fields.
    - HTTP method and path template preserved for the HTTP client in generated code.
    - Response content type (application/json preferred).
  - Deps: T8.1, T2.2

- [x] T8.3 Add unit tests for OpenAPI-to-MCP model converter  Owner: TBD  Est: 1h
  - Acceptance: Tests cover:
    - Operation with path params, query params, and request body
    - Operation with no parameters
    - Operation with no operationId (name derived from method+path)
    - Nested object schemas flattened correctly
    - Enum parameters preserved
    - Multiple security schemes
  - Deps: T8.2

- [x] T8.4 Implement JSON Schema derivation from OpenAPI schemas  Owner: TBD  Est: 1.5h
  - Acceptance: Converts OpenAPI Schema Object to JSON Schema suitable for MCP tool inputSchema. Handles: string, integer, number, boolean, array, object types. Preserves descriptions, enums, defaults, format hints. Resolves `$ref` to inline schemas.
  - Deps: T8.1, T2.2

- [x] T8.5 Add unit tests for JSON Schema derivation  Owner: TBD  Est: 1h
  - Acceptance: Tests cover all primitive types, arrays of objects, nested objects, `$ref` resolution, enum values.
  - Deps: T8.4

- [x] T8.6 Run linter and formatter on E8 code  Owner: TBD  Est: 15m
  - Deps: T8.1 through T8.5

### Epic E9: MCP Server Generation -- Go Output

Generates a complete, deployable Go MCP server project from the MCP model.

- [x] T9.1 Design Go MCP server template structure  Owner: TBD  Est: 1h
  - Acceptance: Document (in code comments or a design doc) describing:
    - Generated file layout: `main.go`, `server.go`, `tools.go`, `client.go`, `types.go`, `go.mod`
    - `main.go`: parses flags (--transport stdio|sse, --port, --api-key), starts server
    - `server.go`: registers all tools, sets up MCP server using `github.com/mark3labs/mcp-go` SDK
    - `tools.go`: one function per tool that makes the HTTP call and returns the result
    - `client.go`: shared HTTP client with auth header injection
    - `types.go`: request/response structs for each operation
    - `go.mod`: module declaration with mcp-go dependency

- [x] T9.2 Create Go MCP server templates using `text/template`  Owner: TBD  Est: 3h
  - Acceptance: Template files in `templates/mcp-go/` embedded via `embed.FS`:
    - `main.go.tmpl`: CLI entry point with transport selection
    - `server.go.tmpl`: MCP server setup, tool registration loop
    - `tools.go.tmpl`: tool handler functions (HTTP request construction, execution, response parsing)
    - `client.go.tmpl`: HTTP client with configurable base URL and auth
    - `types.go.tmpl`: Go structs for request/response bodies
    - `go.mod.tmpl`: module file
    - `README.md.tmpl`: usage instructions for the generated server
  - Each template must produce valid, `gofmt`-clean Go code.
  - Deps: T9.1

- [x] T9.3 Implement Go code generation orchestrator  Owner: TBD  Est: 2h
  - Acceptance: `internal/mcpgen/golang/generate.go` takes an `MCPServer` model and output directory path, executes all templates, writes files to output directory. Returns error if any template fails.
  - Deps: T9.2, T8.2

- [x] T9.4 Implement Go type mapping (OpenAPI types to Go types)  Owner: TBD  Est: 1.5h
  - Acceptance: Maps OpenAPI types to Go types:
    - string -> string
    - integer -> int64
    - number -> float64
    - boolean -> bool
    - array -> []T (recursive)
    - object -> struct with exported fields
    - string with format date-time -> time.Time
    - string with format binary -> []byte
    - nullable types -> pointer types
  - Generates struct definitions with JSON tags.
  - Deps: T8.1

- [x] T9.5 Implement `mint mcp generate` command  Owner: TBD  Est: 1h
  - Acceptance: `mint mcp generate spec.yaml --output ./myserver` produces a directory containing a compilable Go MCP server. `cd myserver && go build ./...` succeeds. No `--lang` flag needed since Go is the only target.
  - Deps: T9.3, T1.3

- [x] T9.6 Add integration tests for Go MCP server generation  Owner: TBD  Est: 2h
  - Acceptance: Tests that:
    - Generated server from petstore spec compiles (`go build`)
    - Generated server starts and responds to MCP initialize request via stdio
    - Generated server lists all expected tools
    - Generated tool input schemas match expected JSON Schema
    - Generated server with auth config includes API key header in HTTP requests
  - Deps: T9.3, T9.5

- [x] T9.7 Run linter and formatter on E9 code  Owner: TBD  Est: 15m
  - Deps: T9.2 through T9.5

### Epic E10: MCP Server Generation -- Advanced Features

- [x] T10.1 Implement authentication passthrough configuration  Owner: TBD  Est: 1.5h
  - Acceptance: Generated servers read API keys from environment variables. Supports three auth patterns:
    - API key in header (`X-API-Key` from `MINT_API_KEY` env var)
    - Bearer token (`Authorization: Bearer` from `MINT_TOKEN` env var)
    - Custom header (configurable name and env var via `--auth-header` and `--auth-env` flags)
  - Deps: T9.3

- [x] T10.2 Implement HTTP/SSE transport support in generated Go servers  Owner: TBD  Est: 2h
  - Acceptance: Generated Go server supports `--transport sse --port 8080` flag. Starts HTTP server with SSE endpoint at `/sse` and message endpoint at `/message`. Conforms to MCP HTTP/SSE transport spec.
  - Deps: T9.3

- [x] T10.3 Implement operation filtering for MCP generation  Owner: TBD  Est: 1h
  - Acceptance: `mint mcp generate --include-tags users,posts` generates tools only for operations tagged with "users" or "posts". `--exclude-paths '/internal/*'` excludes matching paths.
  - Deps: T8.2

- [ ] T10.4 Implement tool name customization via overlay or config  Owner: TBD  Est: 1h
  - Acceptance: Users can provide a mapping file (`mint.yaml`) or overlay that renames tools. Example: map `listPets` to `search_pets`.
  - Deps: T8.2, T6.2

- [x] T10.5 Add Dockerfile template to generated servers  Owner: TBD  Est: 45m
  - Acceptance: Generated Go server includes a multi-stage `Dockerfile` that builds and runs the server. `docker build -t myserver .` succeeds.
  - Deps: T9.3

- [x] T10.6 Add unit and integration tests for advanced MCP features  Owner: TBD  Est: 1.5h
  - Acceptance: Tests cover auth injection, SSE transport startup, operation filtering, tool name customization.
  - Deps: T10.1 through T10.5

- [x] T10.7 Run linter and formatter on E10 code  Owner: TBD  Est: 15m
  - Deps: T10.1 through T10.5

### Epic E11: CI/CD Integration

- [ ] T11.1 Create GitHub Action for mint (validate + diff in PRs)  Owner: TBD  Est: 2h
  - Acceptance: Action installs mint, runs lint, posts results as PR comment. Breaking changes fail the check.
  - Deps: T3.2, T4.2
- [ ] T11.2 Write action.yml and composite action script  Owner: TBD  Est: 1h
  - Acceptance: Valid action.yml with inputs for spec path, ruleset, fail-on-breaking.
  - Deps: T11.1
- [ ] T11.3 Add GitHub Action for MCP server regeneration on spec change  Owner: TBD  Est: 1.5h
  - Acceptance: Action detects spec changes in PR, regenerates MCP server, commits updated code. Configurable via action inputs.
  - Deps: T9.5, T11.1
- [ ] T11.4 Add integration tests for GitHub Actions  Owner: TBD  Est: 1h
  - Acceptance: Actions run successfully in test workflows.
  - Deps: T11.1, T11.2, T11.3
- [ ] T11.5 Run linter and formatter on E11 code  Owner: TBD  Est: 15m
  - Deps: T11.1, T11.2, T11.3

### Epic E12: Documentation and Release

- [x] T12.1 Write README with install instructions, quickstart, and command reference  Owner: TBD  Est: 1.5h
  - Acceptance: README covers all commands with examples. Install via go install, homebrew tap, and binary download. Includes MCP server generation quickstart with Claude Desktop configuration example.
- [x] T12.2 Add CLI help text for every command and flag  Owner: TBD  Est: 1h
  - Acceptance: Every command and subcommand has a usage string. `mint help <cmd>` works for all.
  - Deps: all command tasks
- [x] T12.3 Create CONTRIBUTING.md and LICENSE (Apache 2.0)  Owner: TBD  Est: 30m
- [ ] T12.4 Create example specs and generated servers in `examples/` directory  Owner: TBD  Est: 1.5h
  - Acceptance: At least 3 examples:
    - Petstore spec with generated Go MCP server
    - Multi-file merge example
    - Overlay example
  - Each example includes a README explaining what it demonstrates.
- [ ] T12.5 Write MCP server usage guide (connecting to Claude Desktop, Cursor, etc.)  Owner: TBD  Est: 1h
  - Acceptance: Step-by-step guide: generate server, build, configure in Claude Desktop `claude_desktop_config.json`, test with a prompt.
- [ ] T12.6 Set up homebrew tap for installation  Owner: TBD  Est: 1h
  - Acceptance: `brew install sirerun/tap/<chosen-name>` works on macOS.
  - Deps: T1.7
- [ ] T12.7 Run final linter and formatter pass on entire codebase  Owner: TBD  Est: 30m
  - Deps: all implementation tasks

### Archived

- **E8-v1 (Code Sample Generation)** -- Archived. Reason: Code samples are now embedded within generated MCP server tool handlers rather than being a standalone feature.
- **E9-v1 (SDK Scaffolding)** -- Archived. Reason: Replaced by MCP server generation. Generic SDK scaffolding is out of scope for v1.
- **E10-v2 (TypeScript MCP Server Generation)** -- Archived. Reason: Go is sufficient for v1. TypeScript output may be added in a future version. Reduces scope and eliminates the `@modelcontextprotocol/sdk` dependency from the project.

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M1: Foundation | E1, E2 | None | CLI compiles, loads specs, CI green |
| M2: Core OpenAPI Tools | E3, E4, E5, E6, E7 | M1 | Lint, diff, merge, overlay, transform commands work with JSON output |
| M3: MCP Generation | E8, E9 | M1 | `mint mcp generate petstore.yaml` produces a working Go MCP server |
| M4: MCP Advanced + CI/CD | E10, E11 | M3, M2 | Auth, SSE, filtering complete. GitHub Actions work |
| M5: Ship It | E12 | M2, M3, M4 | README complete, examples published, v0.1.0 released |

### Dependency Graph

```
E1 (Bootstrap) --> E2 (Parsing)
E2 --> E3 (Linting)
E2 --> E4 (Diffing)
E2 --> E5 (Merging)
E2 --> E6 (Overlay)
E2 --> E7 (Transform)
E2 --> E8 (MCP Core Engine)
E8 --> E9 (MCP Go Output)
E9 --> E10 (MCP Advanced Features)
E3, E4 --> E11 (CI/CD)
E9 --> E11 (CI/CD -- MCP regeneration action)
All --> E12 (Docs and Release)
```

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R1 | `pb33f/libopenapi` API changes break parsing | High | Low | Pin dependency version. Wrap in internal adapter. |
| R2 | Subcommand dispatch with `flag` package becomes unwieldy | Medium | Medium | Keep command count small. Use a simple registry pattern. If truly painful, consider migrating to cobra later. |
| R3 | MCP specification changes before v1.0 stable | High | Medium | Target latest stable spec version. Isolate MCP protocol details behind interfaces so updates are localized. |
| R4 | Generated MCP server code quality insufficient for production use | High | Medium | Focus on petstore spec first. Generate clean, readable code. Include error handling and logging. Get early feedback from Claude Desktop users. |
| R5 | `mcp-go` SDK introduces breaking changes | Medium | Medium | Pin SDK version in generated go.mod. Test against specific SDK version. |
| R6 | Complex OpenAPI specs (oneOf, anyOf, discriminators) produce broken MCP tools | High | High | Start with simple specs. Document supported schema features. Degrade gracefully for unsupported patterns (emit warning, use generic JSON type). |
| R7 | Overlay specification is not widely adopted | Low | Medium | Implement anyway -- it is a simple feature and useful for MCP tool customization. |
| R8 | Go code templates become hard to maintain | Medium | Medium | Use `text/template` with clean data models. Keep templates in embedded files. Comprehensive golden-file tests. |

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
7. For MCP generation tasks: generated server code compiles and starts.

### Review and QA Steps

1. Self-review all changed files before marking a task complete.
2. Run `go test ./...` and verify no regressions.
3. Run `golangci-lint run` and fix any findings.
4. Run `gofmt -s -w .` to ensure formatting.
5. Test the affected CLI command manually with at least one real OpenAPI spec.
6. For MCP generation changes: generate a server from petstore spec, build it, and verify it starts and lists tools.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Never allow changes to pile up. Make many small logical commits.
- Each commit should represent one logical change and have a clear message.

---

## Progress Log

### 2026-03-05 -- Change Summary (Update 3)

- Removed Epic E10-v2 (TypeScript MCP Server Generation). Moved to Archived. Go is the only MCP output target for v1.
- Removed deliverable D7 (TypeScript MCP server generator). Renumbered D8-D10 to D7-D9.
- Removed `@modelcontextprotocol/sdk` from dependencies.
- Removed `templates/mcp-ts/` from directory structure (T1.2).
- Simplified T9.5: removed `--lang` flag since Go is the only target.
- Renumbered old E11 (Advanced Features) to E10, old E12 (CI/CD) to E11, old E13 (Docs) to E12.
- Updated T10.1 (auth): removed dependency on T10.3 (TypeScript orchestrator, no longer exists).
- Updated milestones: M4 no longer includes TypeScript generation.
- Added five project name candidates aligned with Sire Run brand identity.
- Updated Non-Goals to explicitly list TypeScript MCP server generation.
- Total: 12 epics, 58 tasks.

### 2026-03-05 -- Change Summary (Update 2)

- Restructured entire plan around MCP server generation as the primary goal.
- Replaced E8 (Code Sample Generation) and E9 (SDK Scaffolding) with new epics:
  - E8: MCP Server Generation -- Core Engine (OpenAPI-to-MCP mapping)
  - E9: MCP Server Generation -- Go Output
  - E10: MCP Server Generation -- TypeScript Output
  - E11: MCP Server Generation -- Advanced Features (auth, SSE, filtering)
- Moved old E8 and E9 to Archived section with reasons.
- Added new deliverables D6 (Go MCP server generator) and D7 (TypeScript MCP server generator).
- Updated milestones: M3 is now MCP Go generation, M4 is MCP TypeScript + advanced features.
- Added risks R3 through R6 for MCP-specific concerns.
- Updated context to position MCP server generation as the primary objective.

### 2026-03-05 -- Change Summary (Update 1)

- Initial plan created with all epics E1 through E11.
- Defined project name: **Mint** (`github.com/sirerun/mint`).
- Identified 11 epics, 56 tasks covering bootstrap through release.
- Established dependency graph and 5 milestones.
- Documented 6 risks with mitigations.

### 2026-03-05 -- Plan Created

- No implementation progress yet. Plan is new.

---

## Hand-off Notes

### What You Need to Know

1. **Project**: An open-source Go CLI whose primary purpose is generating Go MCP servers from OpenAPI specs. Also provides OpenAPI tooling (lint, diff, merge, transform, overlay). Inspired by but not derived from the Speakeasy CLI. No Speakeasy proprietary code is used.
2. **Repo**: `github.com/sirerun/<chosen-name>`. Not yet created. See "Project Name Candidates" section at top of plan.
3. **Key Dependencies**:
   - `pb33f/libopenapi` -- OpenAPI parsing
   - `pb33f/vacuum` -- linting
   - `pb33f/openapi-changes` -- diffing
   - `mark3labs/mcp-go` -- Go MCP SDK (used in generated Go servers, not a build dependency of mint itself)
4. **CLI Framework**: Standard `flag` package with manual subcommand dispatch. No cobra.
5. **Build**: goreleaser for cross-platform releases. GitHub Actions for CI.
6. **MCP Generation**: Uses Go `text/template` with embedded template files. Templates live in `templates/mcp-go/`. The core mapping logic lives in `internal/mcpgen/`.
7. **Testing**: `go test` with standard library. No testify. Test data in `testdata/` directories. Golden-file tests for template output.
8. **Output language**: Go only. TypeScript was considered and explicitly deferred.

### Credentials and Links (Placeholders)

- GitHub org: `github.com/sirerun` -- requires admin access to create repo.
- Homebrew tap: `github.com/sirerun/homebrew-tap` -- to be created.
- No API keys or secrets required for core functionality.

---

## Appendix

### How MCP Server Generation Works

The generation pipeline has three stages:

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
- Verify generated code compiles (optional, via `--verify` flag)

### Example: Petstore to MCP Server

Given a petstore OpenAPI spec with operations `listPets`, `createPet`, `showPetById`, mint generates:

**MCP Tools:**
- `list_pets` -- List all pets. Input: `{ limit?: number }`
- `create_pet` -- Create a pet. Input: `{ name: string, tag?: string }`
- `show_pet_by_id` -- Info for a specific pet. Input: `{ petId: string }`

**Generated Go Server:**
```
myserver/
  main.go          -- entry point, transport selection
  server.go        -- MCP server setup, tool registration
  tools.go         -- list_pets(), create_pet(), show_pet_by_id() handlers
  client.go        -- HTTP client for petstore API
  types.go         -- Pet, Error structs
  go.mod           -- module with mcp-go dependency
  Dockerfile       -- multi-stage build
  README.md        -- usage instructions
```

### UX Improvements Over Speakeasy

1. **No platform lock-in**: Speakeasy requires authentication and communicates with `speakeasyapi.dev`. This tool works fully offline.
2. **Simpler command structure**: Speakeasy has 30+ commands. This tool has a flat, predictable set: lint, diff, merge, overlay, transform, mcp.
3. **Better error messages**: Errors reference spec file paths and line numbers, not internal platform state.
4. **Machine-readable output**: Every command supports `--format json` for CI integration.
5. **No upgrade nag or telemetry**: Does not phone home.
6. **Predictable exit codes**: 0 success, 1 error, 2 breaking changes.

### Key Architecture Decisions

1. **Single binary**: No plugins, no downloaded components. Everything ships in one binary.
2. **Adapter pattern for OpenAPI libraries**: Wrap `pb33f/libopenapi` in an internal adapter so the rest of the codebase does not depend directly on it.
3. **Template-based generation**: Use Go `text/template` with embedded template files in `templates/mcp-go/`.
4. **No global state**: Each command receives its configuration via flags and arguments. No config files required (but supported via `mint.yaml` for project-level defaults like tool name overrides).
5. **MCP model as intermediate representation**: The `internal/mcpgen/model.go` structs are the bridge between OpenAPI parsing and code generation. This decoupling allows adding new output languages later without modifying the parsing or mapping logic.
