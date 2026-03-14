# mint

**The OpenAPI to MCP Platform.**

Turn any API into an MCP server in seconds. Discover, generate, publish, and deploy MCP servers with a single command.

```bash
# 1. Discover a server in the Public Mint Registry
mint registry search stripe

# 2. Install and generate it locally
mint registry install stripe

# 3. Deploy it to the Managed Hosting Platform
mint deploy managed --source ./stripe-mcp
```

## Install

```bash
# Go
go install github.com/sirerun/mint/cmd/mint@latest

# macOS
brew install sirerun/tap/mint

# Binary
# Download from https://github.com/sirerun/mint/releases
```

## Public Mint Registry

The Public Mint Registry is the central hub for discovering and sharing MCP servers. It is a curated collection of thousands of OpenAPI specs that can be instantly generated into ready-to-use MCP servers.

### Discover

```bash
# Search for a server
mint registry search slack

# List all servers in a category
mint registry list --tags payments
```

### Install

Installation downloads the OpenAPI spec and generates a ready-to-use Go MCP server:

```bash
mint registry install stripe --output ./stripe-mcp
```

## Generation

Point mint at any OpenAPI 3.x spec -- local file or URL:

```bash
mint mcp generate https://api.twitter.com/2/openapi.json --output ./twitter-mcp
```

### What Gets Generated

```
twitter-mcp/
  main.go       Entry point with stdio/SSE/CLI mode dispatch
  server.go     MCP server setup and tool registration
  tools.go      One handler per API operation
  cli.go        CLI mode: tools listing, call routing, output formatting
  client.go     HTTP client for the upstream API
  go.mod        Go module with mcp-go dependency
  Dockerfile    Multi-stage distroless build
  README.md     Usage instructions for the generated server
```

Every OpenAPI operation becomes an MCP tool. Operation IDs are mapped to tool names, and summaries become descriptions.

## Deploy

Mint provides multiple deployment paths depending on your needs.

### 1. Managed Hosting

The fastest way to host MCP servers with managed authentication, scaling, and observability.

```bash
# Login
mint login

# Deploy
mint deploy managed --source ./twitter-mcp --public
```

### 2. Self-Hosted (GCP / AWS)

Deploy to your own cloud infrastructure with zero YAML. Mint handles container builds (via Podman), registry setup, IAM roles, and load balancing.

#### Google Cloud Run
```bash
mint deploy gcp --project my-project --source ./twitter-mcp
```

#### AWS ECS Fargate
```bash
mint deploy aws --region us-east-1 --source ./twitter-mcp
```

## Publish & Share

Share your generated MCP servers or raw OpenAPI specs with the community.

```bash
# Login with GitHub
mint login --github your-handle

# Publish your project
mint publish --dir ./my-mcp-server
```

Published servers appear on the registry and can be installed by anyone using `mint registry install`.

## OpenAPI Tooling

Mint includes a full OpenAPI toolkit for preparing specs before generation:

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

## CLI Mode

Generated servers include a built-in CLI for testing and debugging without an MCP client:

```bash
# List all available tools
./twitter-mcp tools

# Call a tool with key=value arguments
./twitter-mcp call find_tweets_by_id ids=1234567890
```

## Requirements

- Go 1.25+
- Podman (for cloud deployments)
- OpenAPI 3.0/3.1 spec (Swagger 2.0 supported via `mint transform convert`)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and architecture overview.

## License

Apache 2.0
