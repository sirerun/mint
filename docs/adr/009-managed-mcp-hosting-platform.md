# ADR 009: Managed MCP Hosting Platform

## Status
Accepted

## Date
2026-03-13

## Context
Mint generates MCP servers and deploys them to user-managed cloud accounts (GCP, AWS, Azure). This requires users to have cloud accounts, credentials, and infrastructure knowledge. A managed hosting option would lower the barrier to zero: users run `mint deploy managed` and get a live URL without any cloud setup.

Key design questions:
- Where does the infrastructure run? Managed cloud accounts.
- How are servers isolated? Per-customer namespaces or per-server containers.
- How is billing handled? Usage-based metering (requests, compute-seconds).
- What is the trust boundary? Users upload generated server code; the platform builds and runs it.

## Decision
Build a managed hosting platform under `mint deploy managed` that:
1. Accepts generated server source via `--source` flag.
2. Builds the container using managed build infrastructure.
3. Deploys to a multi-tenant container platform with per-server isolation.
4. Assigns a `{service}.managed.com` subdomain with managed TLS.
5. Meters usage (requests per month, compute-seconds) for billing.
6. Provides a free tier (1 server, 10k requests/month) and paid tiers.

The backend API runs as a separate service. The mint CLI sends the source tarball to the API, which handles build, deploy, DNS, and metering. The CLI implementation in `internal/deploy/managed/` is a thin client that calls the hosting API.

## Consequences
**Positive:**
- Zero-friction deployment for new users. No cloud account needed.
- Direct revenue path via usage-based billing.
- Every `mint mcp generate` user becomes a potential managed hosting customer.

**Negative:**
- Requires building and operating a hosting platform (infrastructure, billing, support).
- Trust boundary: The platform runs user-generated code. Requires sandboxing and security controls.
- Ongoing infrastructure costs until scale is achieved.
