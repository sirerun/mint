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

## Relationship to the Official MCP Registry

The official MCP registry at `registry.modelcontextprotocol.io` lists pre-built MCP servers (e.g., Brave Search, GitHub, Postgres). These are standalone server packages that users install and run directly.

The mint registry (`sirerun/mcp-registry`) is fundamentally different: it lists OpenAPI specifications that mint can generate MCP servers from. The two registries are complementary, not competing:

- **Official MCP Registry**: curated catalog of pre-built MCP servers. Users install a ready-made binary/package.
- **Mint Registry**: curated catalog of OpenAPI specs. Users run `mint registry install <name>` which fetches the spec and runs `mint mcp generate` locally to produce a custom MCP server.

**Decision: Keep our registry separate (Option 3).** We evaluated three options:
1. Submit generated servers to the official registry -- rejected because our value is on-demand generation, not static binaries.
2. Use the official registry as our backend -- rejected because it does not store OpenAPI spec URLs or generation metadata.
3. Keep separate registries that serve different purposes -- chosen because the data models and user workflows are distinct.

The mint registry may link to official registry entries where a pre-built alternative exists, but the primary value proposition is generating MCP servers from OpenAPI specs that have no pre-built equivalent in the official registry.

## Consequences
**Positive:**
- Removes the "find a spec" barrier. Users get instant value without knowing what OpenAPI is.
- Drives mint installs and brand awareness via a curated, searchable catalog.
- Community contributions via PR keep the registry growing without central effort.
- Complementary to the official MCP registry -- covers APIs that have OpenAPI specs but no pre-built MCP server.

**Negative:**
- Requires ongoing curation to keep spec URLs valid and test compatibility.
- OpenAPI specs change over time; registry entries may go stale without automated freshness checks.
- No pre-built binaries means users still need Go installed for `mint registry install`.
- Users may be confused by two registries; documentation must clearly explain the difference.
