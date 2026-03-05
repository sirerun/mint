# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `mint mcp generate` command to generate Go MCP servers from OpenAPI specs
- `mint validate` command for OpenAPI spec validation
- Generated servers support stdio and SSE transports via `--transport` flag
- Authentication passthrough (API key, Bearer token, custom header)
- Operation filtering with `--include-tags` and `--exclude-paths`
- Dockerfile included in generated servers
- JSON output format for validation results (`--format json`)
- Cross-platform binary releases via goreleaser
- GitHub Actions CI pipeline
