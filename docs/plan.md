# Mint -- CLI Mode for Generated MCP Servers

## Context

See docs/design.md for full project context, architecture, and conventions.

### Problem Statement

Mint-generated MCP servers currently support two transport modes: stdio (JSON-RPC over stdin/stdout for MCP clients) and SSE (HTTP Server-Sent Events). Both modes require an MCP client to interact with the server. There is no way for a user to invoke a tool directly from the terminal as a one-shot command.

Users want to test and debug generated servers without setting up an MCP client. They want to run commands like:

```
./server call list_pets limit=10
./server tools
```

This requires a third mode in the generated `main.go` that parses CLI arguments, invokes the appropriate MCP tool handler directly, and prints human-readable output to stdout.

### Objectives

1. Generated MCP servers support a `call` subcommand that invokes any registered tool with key=value arguments.
2. Generated MCP servers support a `tools` subcommand that lists all available tools with descriptions and parameters.
3. Output is human-readable by default (formatted JSON), with a `--raw` flag for compact JSON.
4. The CLI mode does not require an MCP client -- it calls the tool handler functions directly.

### Non-Goals

- Interactive REPL mode.
- Shell completions for tool names.
- Streaming responses in CLI mode.

### Constraints and Assumptions

- Generated servers use the standard `flag` package, not cobra. CLI subcommands are dispatched via positional arguments after flag parsing.
- The `call` subcommand must work with the same Server struct and handler methods used by stdio/sse modes.
- Arguments are passed as `key=value` pairs (positional args after the tool name), not as `--key value` flags, because tool parameter names are dynamic and not known at compile time.

### Success Metrics

- `./server tools` lists all tools with their descriptions and parameter schemas.
- `./server call <tool_name> key=value ...` invokes the tool and prints formatted JSON output.
- `./server call <tool_name> --raw key=value ...` prints compact JSON.
- `./server call <unknown_tool>` prints an error listing available tools.
- Existing stdio and sse modes are unaffected.

---

## Scope and Deliverables

### In Scope

- New `cli.go.tmpl` template for CLI dispatch logic (tools listing, call routing, argument parsing, output formatting).
- Updated `main.go.tmpl` to dispatch `tools` and `call` subcommands before the transport switch.
- Updated `readme.md.tmpl` with CLI usage examples.
- Unit tests for the CLI dispatch logic in the template test suite.
- An integration test that generates a server from the petstore spec, builds it, and runs `tools` and `call` subcommands.

### Out of Scope

- Changes to the MCP model, converter, or existing transport templates (server.go, tools.go, client.go).
- Changes to the mint CLI itself (cmd/mint/).

### Deliverables

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D13 | CLI mode for generated servers | TBD | `./server tools` and `./server call <tool> key=value` work on petstore-generated server |

---

## Checkable Work Breakdown

### Epic E41: CLI Mode Template

- [x] T41.1 Create cli.go.tmpl template  Owner: TBD  Est: 1h  Done: 2026-03-13
  - Dependencies: none
  - AC: Template generates a `cli.go` file with three exported functions on Server:
    - `RunCLI(args []string) error` -- entry point that dispatches to tools or call.
    - `ListTools(w io.Writer)` -- prints table of tool names, descriptions, and parameters to w.
    - `CallTool(ctx context.Context, name string, args map[string]interface{}, raw bool, w io.Writer) error` -- builds an `mcp.CallToolRequest`, calls the matching tool handler, formats and prints the result to w.
  - `CallTool` must parse key=value strings into a `map[string]interface{}`, attempting numeric conversion for values that look like numbers (so `limit=10` passes as float64, matching JSON semantics).
  - If the tool name is not found, print an error with the list of available tools.
  - Risk: The `mcp.CallToolRequest` construction must match what `mcp-go` expects internally.

- [x] T41.2 Update main.go.tmpl for subcommand dispatch  Owner: TBD  Est: 30m  Done: 2026-03-13
  - Dependencies: T41.1
  - AC: Before flag parsing and the transport switch, check `os.Args` for subcommands:
    - `os.Args[1] == "tools"` -> call `srv.ListTools(os.Stdout)` and exit.
    - `os.Args[1] == "call"` -> call `srv.RunCLI(os.Args[2:])` and exit.
    - Otherwise, proceed to existing flag parsing and transport switch.
  - The subcommand check must happen before `flag.Parse()` because `flag.Parse()` would reject unknown positional args.

- [x] T41.3 Register cli.go.tmpl in generate.go  Owner: TBD  Est: 15m  Done: 2026-03-13
  - Dependencies: T41.1
  - AC: Add `{"templates/cli.go.tmpl", "cli.go"}` to the templates slice in `Generate()`. The generated cli.go compiles with the rest of the project.

- [x] T41.4 Update readme.md.tmpl with CLI usage  Owner: TBD  Est: 15m  Done: 2026-03-13
  - Dependencies: T41.1
  - AC: Add a "CLI Mode" section showing `tools` and `call` usage examples.

- [x] T41.5 Add unit tests for CLI template generation  Owner: TBD  Est: 45m  Done: 2026-03-13
  - Dependencies: T41.1, T41.3
  - AC: Test in `internal/mcpgen/golang/generate_test.go` (or a new `cli_test.go`):
    - Generate from petstore spec and verify `cli.go` exists in output.
    - Verify `cli.go` contains expected function signatures.
    - Verify `main.go` contains subcommand dispatch code.

- [x] T41.6 Add integration test: build and run CLI mode  Owner: TBD  Est: 1h  Done: 2026-03-13
  - Dependencies: T41.1, T41.2, T41.3
  - AC: Integration test that:
    1. Generates a server from `testdata/petstore.yaml`.
    2. Runs `go build` on the output.
    3. Runs `./server tools` and verifies tool names appear in output.
    4. Runs `./server call list_pets limit=10` and verifies JSON output (the API call will fail since there is no real server, but the argument parsing and dispatch should work -- verify the error message mentions the HTTP call, not a CLI parsing error).
    5. Runs `./server call nonexistent_tool` and verifies error output lists available tools.

- [x] T41.7 Run linter and formatter  Owner: TBD  Est: 15m  Done: 2026-03-13
  - Dependencies: T41.1 through T41.6
  - AC: `golangci-lint run` and `gofmt -s` produce no findings on changed files.

---

## Parallel Work

| Track | Task IDs | Description |
|-------|----------|-------------|
| Track A: Templates | T41.1, T41.2, T41.3, T41.4 | Create and wire CLI templates |
| Track B: Tests | T41.5, T41.6 | Unit and integration tests |

Track A must complete before Track B starts.

### Maximum Parallelism

| Wave | Tasks | Notes |
|------|-------|-------|
| Wave 1 | T41.1 | Core template -- all other tasks depend on this |
| Wave 2 | T41.2, T41.3, T41.4 | Can run in parallel once T41.1 exists |
| Wave 3 | T41.5, T41.6 | Tests run in parallel after templates are wired |
| Wave 4 | T41.7 | Lint/format after all code is written |

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M26: CLI Templates | T41.1-T41.4 | none | Generated server compiles with CLI mode |
| M27: CLI Validated | T41.5-T41.7 | M26 | All tests pass, lint clean |

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R14 | mcp.CallToolRequest internal structure changes between mcp-go versions | CLI call dispatch breaks | Low | Pin mcp-go version in generated go.mod. Use only public API. |
| R15 | key=value parsing fails for complex values (JSON objects, arrays) | Some tools unusable from CLI | Medium | Support `key=@file.json` syntax for complex values in a follow-up if needed. Document limitation. |

---

## Operating Procedure

### Definition of Done

A task is done when:
1. Code compiles with zero warnings.
2. All new code has unit tests with 100% coverage.
3. `go test ./... -race` passes with no regressions.
4. `golangci-lint run` passes with no new findings.
5. `gofmt -s` produces no changes.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Conventional Commits: feat(mcpgen):, test(mcpgen):.

---

## Progress Log

### 2026-03-13 -- Plan Created

Created plan for CLI mode in generated MCP servers. Defined epic E41 with 7 tasks. The feature adds a `call` subcommand and `tools` listing to generated servers, enabling direct terminal usage without an MCP client. Implementation is entirely within the template layer (new cli.go.tmpl, updated main.go.tmpl and readme.md.tmpl).

Prior E2E validation work (E29) is tracked separately and remains blocked on cloud sandbox credentials.

---

## Hand-off Notes

- This feature only changes generated server templates in `internal/mcpgen/golang/templates/`. No changes to the mint CLI itself.
- The key design choice is dispatching subcommands via positional args (`os.Args[1]`) before `flag.Parse()`, since tool parameter names are dynamic.
- Arguments use `key=value` syntax (not `--key value`) because tool parameters are not known at compile time.
- The `CallTool` function constructs an `mcp.CallToolRequest` and calls the handler directly -- it does not go through the stdio or SSE transport.
- Existing template tests are in `internal/mcpgen/golang/generate_test.go`.
- Petstore test spec is at `testdata/petstore.yaml`.

---

## Appendix

### Example CLI Usage (Generated Server)

```
# List all available tools
./server tools

# Call a tool with arguments
./server call list_pets limit=10

# Call with raw (compact) JSON output
./server call list_pets --raw limit=10

# Call a tool with no arguments
./server call get_server_info

# Error: unknown tool
./server call nonexistent
Error: unknown tool "nonexistent". Available tools:
  list_pets      - List all pets
  create_pet     - Create a new pet
  get_pet_by_id  - Get a pet by its ID
```

### cli.go.tmpl Sketch

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "strconv"
    "strings"
    "text/tabwriter"

    "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) RunCLI(args []string) error {
    if len(args) == 0 {
        s.ListTools(os.Stdout)
        return nil
    }

    toolName := args[0]
    raw := false
    kvArgs := args[1:]

    // Check for --raw flag
    for i, a := range kvArgs {
        if a == "--raw" {
            raw = true
            kvArgs = append(kvArgs[:i], kvArgs[i+1:]...)
            break
        }
    }

    params := make(map[string]interface{})
    for _, kv := range kvArgs {
        k, v, ok := strings.Cut(kv, "=")
        if !ok {
            return fmt.Errorf("invalid argument %q: expected key=value", kv)
        }
        // Try numeric conversion
        if n, err := strconv.ParseFloat(v, 64); err == nil {
            params[k] = n
        } else if v == "true" || v == "false" {
            params[k] = v == "true"
        } else {
            params[k] = v
        }
    }

    return s.CallTool(context.Background(), toolName, params, raw, os.Stdout)
}
```
