# ADR 003: AI-Native Release Pipeline Design

## Status
Accepted

## Date
2026-03-07

## Context
Mint targets AI-native development workflows where features are released at the speed of thought. The release pipeline must support extremely high deployment frequency -- potentially multiple deployments per hour triggered by AI agents or developers using AI coding assistants.

Traditional CI/CD pipelines with manual approval gates, long test cycles, and sequential stages are too slow for this workflow. The pipeline must be optimized for:
1. Sub-minute build and deploy cycles.
2. Automated rollback on failure.
3. Zero-downtime deployments.
4. No manual gates between code change and production (for non-breaking changes).

## Decision
Design the release pipeline with the following properties:

1. **Single-command deploy**: `mint deploy gcp` handles build, push, and deploy in one invocation.
2. **Cloud Build for container images**: No local Docker required. Cloud Build produces images in under 60 seconds for Go binaries.
3. **Cloud Run revision-based deployments**: Each deploy creates a new revision. Traffic shifts instantly. Previous revision remains available for rollback.
4. **Automated health checks**: After deployment, verify the new revision responds to MCP initialize requests. Roll back automatically if health check fails.
5. **Canary deployments (optional)**: `--canary` flag routes a percentage of traffic to the new revision before full cutover.
6. **GitHub Actions integration**: Workflow template that triggers deploy on push to main or on spec file changes.
7. **Idempotent deploys**: Running `mint deploy gcp` twice with the same code produces the same result. Safe for AI agents to invoke repeatedly.

## Consequences
**Positive:**
- Developers and AI agents can deploy with a single command.
- Rollback is instant (revert to previous Cloud Run revision).
- No local Docker installation required.
- Pipeline supports both human and AI-driven release workflows.

**Negative:**
- Fast deployment without manual gates increases risk of deploying broken code. Mitigated by automated health checks and instant rollback.
- Cloud Build adds GCP cost per build. Mitigated by scale-to-zero and short build times for Go.
- Canary deployments add complexity. Made optional via flag.
