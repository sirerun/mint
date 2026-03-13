# ADR 010: MCP Server Registry

## Status
Accepted

## Date
2026-03-13

## Context
Users must currently find an OpenAPI spec before they can generate an MCP server with mint. This is a significant barrier -- not all APIs publish OpenAPI specs, and finding the right spec URL requires research. A registry of pre-built, versioned MCP servers from popular APIs would provide instant value.

Key design questions:
- Where are registry entries stored? A public GitHub repo or a dedicated API.
- What does an entry contain? Metadata (API name, description, version, spec URL) and optionally pre-built binaries.
- How are entries curated? Community contributions via PR, automated spec discovery, or manual curation.
- How does the CLI interact? `mint registry search`, `mint registry install`.

## Decision
Build a public MCP server registry with two components:
1. **Registry index**: A JSON file hosted in a public GitHub repo (`sirerun/mcp-registry`) containing metadata for each registered API: name, description, tags, OpenAPI spec URL, auth requirements, and latest tested mint version.
2. **CLI commands**: `mint registry search <query>`, `mint registry list`, `mint registry install <name>` (generates + builds the server locally).

The registry does NOT host pre-built binaries. Instead, `mint registry install` fetches the OpenAPI spec and runs `mint mcp generate` locally. This avoids supply chain risks and keeps the registry lightweight.

Initial seed: 20-30 popular APIs (Twitter/X, GitHub, Stripe, Slack, OpenAI, etc.) with verified OpenAPI specs.

## Consequences
**Positive:**
- Removes the "find a spec" barrier. Users get instant value without knowing what OpenAPI is.
- Drives mint installs and brand awareness via a curated, searchable catalog.
- Community contributions via PR keep the registry growing without central effort.

**Negative:**
- Requires ongoing curation to keep spec URLs valid and test compatibility.
- OpenAPI specs change over time; registry entries may go stale without automated freshness checks.
- No pre-built binaries means users still need Go installed for `mint registry install`.
