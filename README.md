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
| `--tool-names` | | YAML file mapping original tool names to custom names |

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

# Custom tool names
cat > tool-names.yaml << 'EOF'
list_pets: get_all_animals
create_pet: add_animal
EOF
mint mcp generate --tool-names tool-names.yaml --output ./server spec.yaml
```

### `mint lint`

Lint an OpenAPI spec with configurable rulesets.

```bash
mint lint [flags] <spec-file>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text` or `json` |
| `--ruleset` | `recommended` | Ruleset: `minimal`, `recommended`, or `strict` |

**Examples:**

```bash
# Lint with recommended rules
mint lint petstore.yaml

# Strict ruleset
mint lint --ruleset strict petstore.yaml

# JSON output for CI
mint lint --format json petstore.yaml
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

### `mint diff`

Compare two OpenAPI specs and detect breaking changes.

```bash
mint diff [flags] <old-spec> <new-spec>
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `text` | Output format: `text` or `json` |
| `--fail-on-breaking` | `false` | Exit with code 1 if breaking changes found |

**Examples:**

```bash
# Compare two specs
mint diff old-api.yaml new-api.yaml

# Fail CI on breaking changes
mint diff --fail-on-breaking old-api.yaml new-api.yaml
```

### `mint merge`

Merge multiple OpenAPI specs into one.

```bash
mint merge [flags] <spec1> <spec2> [spec3...]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-o` | stdout | Output file |
| `--on-conflict` | `fail` | Conflict strategy: `fail`, `skip`, or `rename` |

**Examples:**

```bash
# Merge two specs
mint merge users-api.yaml pets-api.yaml -o merged.yaml

# Skip conflicting paths
mint merge --on-conflict skip api-v1.yaml api-v2.yaml -o merged.yaml
```

### `mint overlay apply`

Apply an OpenAPI Overlay document to a spec.

```bash
mint overlay apply [flags] <spec-file> <overlay-file>
```

### `mint transform`

Transform OpenAPI specs.

```bash
# Filter operations by tags
mint transform filter --tags pets spec.yaml

# Remove unused components
mint transform cleanup spec.yaml -o cleaned.yaml

# Normalize/format a spec
mint transform format spec.yaml -o formatted.yaml

# Convert Swagger 2.0 to OpenAPI 3.0
mint transform convert swagger2.yaml -o openapi3.yaml
```

## GitHub Action

Use mint in your CI/CD pipelines:

```yaml
- uses: sirerun/mint@main
  with:
    spec: "api/openapi.yaml"
    command: "lint"
    ruleset: "recommended"
```

See [`.github/workflows/mint-example.yml`](.github/workflows/mint-example.yml) for a complete example.

## Deploying to Cloud Run

Deploy a generated MCP server to Google Cloud Run with a single command. Deployments enforce SOC2-compliant security defaults out of the box.

### Prerequisites

- A GCP project with billing enabled.
- The `gcloud` CLI installed for initial authentication only:

```bash
gcloud auth application-default login
```

The GCP Go SDK handles all provisioning. No Terraform or `gcloud` commands are used after authentication.

### Quickstart

```bash
# Generate an MCP server from an OpenAPI spec
mint mcp generate petstore.yaml --output ./server

# Deploy to Cloud Run
mint deploy gcp --project my-project --region us-central1 --source ./server
```

The deploy command builds a container image via Cloud Build, pushes it to Artifact Registry, provisions a Cloud Run service, verifies health, and prints the live endpoint URL.

### Security Controls

All deployments enforce the following by default:

- **IAM authentication** -- no unauthenticated access unless `--public` is explicitly set.
- **Distroless containers** -- minimal attack surface, no shell in the image.
- **Non-root execution** -- container runs as `nonroot` user.
- **TLS 1.2+** -- enforced by Cloud Run's built-in TLS termination.
- **Secret Manager** -- secrets are mounted as environment variables, never baked into images.
- **Audit metadata** -- every revision is labeled with commit SHA, spec hash, deployer, and timestamp.

### Secrets Management

Mount secrets from GCP Secret Manager as environment variables using the `--secret` flag:

```bash
mint deploy gcp --project my-project --source ./server \
  --secret API_KEY=petstore-api-key \
  --secret DB_PASSWORD=petstore-db-pass
```

Secrets are created in Secret Manager if they do not already exist. Set their values via the GCP console or `gcloud`. The service account is automatically granted `secretAccessor` on each secret.

### Canary Deployments

Roll out gradually by sending a percentage of traffic to the new revision:

```bash
# Deploy with 10% traffic to the new revision
mint deploy gcp --project my-project --source ./server --canary 10

# After validation, promote to 100%
mint deploy gcp --promote --project my-project --service petstore-mcp
```

### CI/CD with GitHub Actions

Generate a deploy-on-push workflow and provision Workload Identity Federation for keyless GCP authentication:

```bash
mint deploy gcp --project my-project --source ./server --ci
```

This creates `.github/workflows/deploy-gcp.yml` and configures a Workload Identity Pool and Provider linked to your GitHub repository. No service account keys are needed.

### Status and Rollback

```bash
# Check deployment status
mint deploy status --project my-project --service petstore-mcp

# JSON output for scripting
mint deploy status --project my-project --service petstore-mcp --format json

# Rollback to the previous revision
mint deploy rollback --project my-project --service petstore-mcp
```

Rollback shifts 100% of traffic to the previous revision. Automatic rollback also triggers if the post-deploy health check fails.

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
- An OpenAPI 3.0/3.1 specification (or Swagger 2.0 — use `mint transform convert` first)

## License

Apache 2.0 - see [LICENSE](LICENSE)
