# mint

Generate production-ready MCP servers from OpenAPI specs with a single command.

**mint** is an open-source Go CLI that turns any OpenAPI 3.0/3.1 specification into a working Go MCP server. The generated server uses the [mcp-go](https://github.com/mark3labs/mcp-go) SDK and supports both stdio and SSE transports.

## Install

### From source

```bash
go install github.com/sirerun/mint/cmd/mint@latest
```

### From binary releases

Download the latest release from [GitHub Releases](https://github.com/sirerun/mint/releases).

## Quick Start

### Generate an MCP server

```bash
mint mcp generate --output ./myserver petstore.yaml
```

### Build and run

```bash
cd myserver
go mod tidy
go build -o myserver .

# Run with stdio transport (for Claude Desktop, Cursor, etc.)
./myserver --transport stdio

# Run with SSE transport
./myserver --transport sse --port 8080
```

### Connect to Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "myserver": {
      "command": "/path/to/myserver",
      "args": ["--transport", "stdio"]
    }
  }
}
```

## Commands

### `mint mcp generate`

Generate a Go MCP server from an OpenAPI spec.

```bash
mint mcp generate [flags] <spec-file>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `./server` | Output directory |
| `--include-tags` | | Only include operations with these tags (comma-separated) |
| `--exclude-paths` | | Exclude paths matching patterns (comma-separated, supports `*` suffix) |
| `--auth-header` | | Override auth header name from spec |
| `--auth-env` | | Override env var name for auth token |

**Examples:**

```bash
# Generate from a local file
mint mcp generate --output ./petstore-server petstore.yaml

# Only include specific tags
mint mcp generate --include-tags users,pets --output ./api-server spec.yaml

# Exclude internal endpoints
mint mcp generate --exclude-paths '/internal/*,/admin/*' --output ./public-server spec.yaml

# Custom auth configuration
mint mcp generate --auth-header X-Custom-Key --auth-env MY_API_KEY --output ./server spec.yaml
```

### `mint validate`

Validate an OpenAPI spec for structural correctness.

```bash
mint validate [flags] <spec-file>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text` or `json` |

**Examples:**

```bash
# Text output
mint validate petstore.yaml

# JSON output for CI
mint validate --format json petstore.yaml
```

## What Gets Generated

Given an OpenAPI spec, mint generates a complete Go project:

```
myserver/
  main.go          # CLI entry point with transport selection
  server.go        # MCP server setup, tool registration
  tools.go         # Tool handler functions (one per operation)
  client.go        # HTTP client for the upstream API
  go.mod           # Go module with mcp-go dependency
  Dockerfile       # Multi-stage Docker build
  README.md        # Usage instructions
```

Each OpenAPI operation becomes an MCP tool:
- **Tool name**: derived from `operationId` (converted to snake_case)
- **Tool description**: from operation `summary` or `description`
- **Input schema**: JSON Schema combining path, query, and body parameters
- **Handler**: makes the HTTP call to the upstream API and returns the response

## How It Works

1. **Parse**: Load the OpenAPI spec using [libopenapi](https://github.com/pb33f/libopenapi)
2. **Map**: Convert each operation to an MCP tool with typed input schema
3. **Generate**: Execute Go templates to produce the server code
4. **Build**: The generated code compiles with `go build` and runs immediately

## Authentication

mint detects security schemes from the OpenAPI spec:

- **API Key**: reads from `MINT_API_KEY` env var, sends as configured header
- **Bearer Token**: reads from `MINT_TOKEN` env var, sends as `Authorization: Bearer`
- **OAuth2**: reads from `MINT_TOKEN` env var

Override with `--auth-header` and `--auth-env` flags.

## Requirements

- Go 1.23+ (for building generated servers)
- An OpenAPI 3.0 or 3.1 specification

## License

Apache 2.0 - see [LICENSE](LICENSE)
