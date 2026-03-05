# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `mint mcp generate` command to generate Go MCP servers from OpenAPI specs
- `mint validate` command for OpenAPI spec validation
- `mint diff` command to compare two specs with breaking change detection
- `mint merge` command to merge multiple specs with conflict strategies
- `mint overlay apply` command for OpenAPI Overlay specification support
- `mint transform filter` command to filter operations by tags or paths
- `mint transform cleanup` command to remove unused components
- `mint transform format` command to normalize and sort spec keys
- Generated servers support stdio and SSE transports via `--transport` flag
- Authentication passthrough (API key, Bearer token, custom header)
- Operation filtering with `--include-tags` and `--exclude-paths`
- Dockerfile included in generated servers
- JSON output format for validation and diff results (`--format json`)
- `mint lint` command with configurable rulesets (minimal, recommended, strict)
- `mint transform convert` command for Swagger 2.0 to OpenAPI 3.0 conversion
- Colored terminal output for lint and validate commands (auto-disabled when not a TTY)
- Tool name customization via `--tool-names` YAML mapping file
- GitHub Action for running mint in CI/CD pipelines (`sirerun/mint@main`)
- Cross-platform binary releases via goreleaser
- GitHub Actions CI pipeline
