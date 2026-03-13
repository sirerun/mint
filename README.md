# mint

Turn any API into an MCP server in seconds.

```bash
mint mcp generate https://api.twitter.com/2/openapi.json
```

That's it. You now have a production-ready Go MCP server with 156 tools, stdio and SSE transports, health checks, and a Dockerfile. Connect it to Claude, Cursor, or any MCP client immediately.

## Install

```bash
# Go
go install github.com/sirerun/mint/cmd/mint@latest

# macOS
brew install sirerun/tap/mint

# Binary
# Download from https://github.com/sirerun/mint/releases
```

## Quick Start

### 1. Generate

Point mint at any OpenAPI 3.x spec -- local file or URL:

```bash
mint mcp generate https://api.twitter.com/2/openapi.json --output ./twitter-mcp
```

### 2. Build and run

```bash
cd twitter-mcp
go mod tidy && go build -o twitter-mcp .

# stdio transport (Claude Desktop, Cursor, Windsurf)
./twitter-mcp --transport stdio

# SSE transport (remote clients, Cloud Run)
./twitter-mcp --transport sse --port 8080
```

### 3. Connect to Claude Desktop

```json
{
  "mcpServers": {
    "twitter": {
      "command": "/path/to/twitter-mcp",
      "args": ["--transport", "stdio"]
    }
  }
}
```

### 4. Deploy to Cloud Run

```bash
gcloud auth application-default login
mint deploy gcp --project my-project --source ./twitter-mcp
```

One command. Builds the container, pushes to Artifact Registry, deploys to Cloud Run, verifies health, prints the live URL. No Dockerfile edits, no `gcloud run deploy`, no YAML.

## What Gets Generated

```
twitter-mcp/
  main.go       Entry point with stdio/SSE transport selection
  server.go     MCP server setup and tool registration
  tools.go      One handler per API operation
  client.go     HTTP client for the upstream API
  go.mod        Go module with mcp-go dependency
  Dockerfile    Multi-stage distroless build
  README.md     Usage instructions for the generated server
```

Every OpenAPI operation becomes an MCP tool:

| OpenAPI | MCP Tool |
|---------|----------|
| `operationId` | Tool name (snake_case) |
| `summary` | Tool description |
| Path + query + body params | JSON Schema `inputSchema` |
| Security schemes | Auth via environment variables |

## OpenAPI Tooling

mint includes a full OpenAPI toolkit for preparing specs before generation:

```bash
# Validate structure
mint validate api.yaml

# Lint with configurable rulesets
mint lint --ruleset strict api.yaml

# Detect breaking changes between versions
mint diff --fail-on-breaking old.yaml new.yaml

# Merge multiple specs
mint merge users.yaml billing.yaml -o combined.yaml

# Apply an OpenAPI Overlay
mint overlay apply api.yaml overlay.yaml

# Filter, clean up, format, or convert Swagger 2.0
mint transform filter --tags users api.yaml
mint transform convert swagger2.yaml -o openapi3.yaml
```

All commands support `--format json` for CI integration.

## Deploy to Cloud Run

Deploy generated MCP servers to Google Cloud Run with security defaults enforced out of the box.

```bash
# Basic deploy
mint deploy gcp --project my-project --source ./server

# With secrets from Secret Manager
mint deploy gcp --project my-project --source ./server \
  --secret API_KEY=my-api-key \
  --secret DB_PASSWORD=my-db-pass

# Canary rollout (10% traffic to new revision)
mint deploy gcp --project my-project --source ./server --canary 10

# Promote canary to 100%
mint deploy gcp --promote --project my-project --service my-server

# Check status
mint deploy status --project my-project --service my-server

# Rollback to previous revision
mint deploy rollback --project my-project --service my-server

# Generate GitHub Actions workflow with Workload Identity Federation
mint deploy gcp --project my-project --source ./server --ci
```

**Security defaults** -- every deployment gets IAM authentication, distroless containers, non-root execution, TLS 1.2+, and Secret Manager integration. No configuration needed.

## Generation Options

```bash
mint mcp generate [flags] <spec-file-or-url>
```

| Flag | Description |
|------|-------------|
| `--output` | Output directory (default: `./server`) |
| `--include-tags` | Only include operations with these tags |
| `--exclude-paths` | Exclude paths matching patterns (supports `*`) |
| `--auth-header` | Override auth header name |
| `--auth-env` | Override env var for auth token |
| `--tool-names` | YAML file mapping tool names to custom names |

```bash
# Generate only user-related endpoints
mint mcp generate --include-tags users --output ./users-mcp api.yaml

# Exclude internal endpoints
mint mcp generate --exclude-paths '/internal/*,/admin/*' api.yaml

# Custom auth
mint mcp generate --auth-header X-Api-Key --auth-env MY_KEY api.yaml
```

## Authentication

mint reads security schemes from the OpenAPI spec and wires them automatically:

| Scheme | Environment Variable | Header |
|--------|---------------------|--------|
| API Key | `MINT_API_KEY` | From spec |
| Bearer / OAuth2 | `MINT_TOKEN` | `Authorization: Bearer` |

Override with `--auth-header` and `--auth-env`.

## GitHub Action

```yaml
- uses: sirerun/mint@main
  with:
    spec: "api/openapi.yaml"
    command: "lint"
    ruleset: "recommended"
```

## Requirements

- Go 1.23+
- OpenAPI 3.0/3.1 spec (Swagger 2.0 supported via `mint transform convert`)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

Apache 2.0
