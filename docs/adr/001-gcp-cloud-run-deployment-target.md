# ADR 001: GCP Cloud Run as Initial Deployment Target

## Status
Accepted

## Date
2026-03-07

## Context
Mint generates Go MCP servers from OpenAPI specs. Users need a way to deploy these servers to production with a single command. The first cloud target must be chosen. Options considered: AWS Lambda/ECS/Fargate, GCP Cloud Run, Azure Container Apps.

Cloud Run is a serverless container platform that supports HTTP and gRPC, scales to zero, and has built-in SOC2/ISO 27001/FedRAMP compliance. MCP servers use HTTP/SSE transport which maps directly to Cloud Run's HTTP-based model. Cloud Run also supports always-on instances for persistent SSE connections.

## Decision
Use Google Cloud Run as the initial deployment target for `mint deploy`. Cloud Run is chosen because:

1. Native HTTP container support aligns with MCP HTTP/SSE transport.
2. SOC2 Type II, ISO 27001, FedRAMP compliance built into the platform.
3. Scale-to-zero reduces cost for low-traffic MCP servers.
4. Minimal configuration needed -- no cluster management, no VPC required for basic deployments.
5. Cloud Build integration for container image building without local Docker.
6. Artifact Registry for container image storage with vulnerability scanning.

## Consequences
**Positive:**
- Users deploy MCP servers with `mint deploy gcp` without managing infrastructure.
- SOC2 compliance is inherited from the platform with correct IAM and encryption settings.
- Future cloud providers (AWS, Azure) can be added as additional subcommands.

**Negative:**
- Users must have a GCP project and billing account.
- SSE transport requires configuring Cloud Run for longer request timeouts (up to 60 minutes).
- Vendor-specific code in `internal/deploy/gcp/` package.
