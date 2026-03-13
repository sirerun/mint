# Mint -- Phase 2 Remaining: E2E Validation

## Context

See docs/design.md for full project context, architecture, and conventions.

### Problem Statement

Mint deploys generated MCP servers to GCP Cloud Run, AWS ECS Fargate, and Azure Container Apps. All three providers have complete implementations with unit tests, CLI wiring, CI workflow generation, and production hardening (auto-scaling, custom domains, graceful shutdown, observability). The remaining work is E2E validation against real cloud accounts.

Additionally, the mint registry at `sirerun/mcp-registry` is a curated catalog of OpenAPI specs (separate from the official MCP registry at `registry.modelcontextprotocol.io` which lists pre-built servers). See docs/adr/010-mcp-server-registry.md for the full rationale.

### Objectives

1. Validate all three deploy providers end-to-end with real cloud accounts.
2. Fix any bugs discovered during E2E testing.

### Non-Goals

- New feature development (all features are implemented).
- Registry changes (Option 3 decision is finalized in ADR 010).

### Constraints and Assumptions

- E2E validation requires real AWS and Azure sandbox credentials.
- GCP E2E was validated in M13 with the Twitter API v2 spec.
- Azure E2E depends on Azure CLI wiring (E33, completed).

### Success Metrics

- Twitter API v2 MCP server deploys and responds on AWS and Azure.
- Canary deployment works on both providers.
- All bugs found during E2E are fixed and committed.

---

## Scope and Deliverables

### In Scope

- E2E validation for AWS and Azure against real cloud accounts.
- Bug fixes discovered during validation.

### Out of Scope

- New provider targets or features.
- Registry UI or changes to registry architecture.

### Deliverables

| ID | Description | Owner | Acceptance Criteria |
|----|-------------|-------|---------------------|
| D12 | E2E validation (AWS + Azure) | TBD | Twitter MCP server deploys and responds on AWS and Azure |

---

## Checkable Work Breakdown

### Epic E29: E2E Validation

- [ ] T29.1 Deploy Twitter API v2 MCP server to AWS  Owner: TBD  Est: 2h
  - Dependencies: none (AWS deploy is complete)
  - AC: Generate MCP server from Twitter API v2 spec. Deploy to ECS Fargate in AWS sandbox. `curl /health` returns 200. Status and rollback commands work.

- [ ] T29.2 Validate canary deployment on AWS  Owner: TBD  Est: 1h
  - Dependencies: T29.1
  - AC: Deploy with `--canary 20`, verify ALB routes 20% to new target group. `--promote` shifts to 100%.

- [ ] T29.3 Deploy Twitter API v2 MCP server to Azure  Owner: TBD  Est: 2h
  - Dependencies: none (Azure deploy is complete)
  - AC: Deploy to Azure Container Apps. Health check passes. Status and rollback work.

- [ ] T29.4 Validate canary deployment on Azure  Owner: TBD  Est: 1h
  - Dependencies: T29.3
  - AC: Deploy with `--canary 20`, verify revision traffic split. Promote shifts to 100%.

- [ ] T29.5 Document and fix all bugs found during E2E  Owner: TBD  Est: 2h
  - Dependencies: T29.1 through T29.4
  - AC: All bugs fixed and committed.

---

## Parallel Work

| Track | Task IDs | Description |
|-------|----------|-------------|
| Track A: AWS E2E | T29.1, T29.2 | AWS deploy + canary validation |
| Track B: Azure E2E | T29.3, T29.4 | Azure deploy + canary validation |

Track A and Track B can run in parallel. T29.5 (bug fixes) runs after both tracks complete.

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M24: All Providers Validated | E29 | none | E2E passes on AWS and Azure with Twitter MCP server |
| M25: Production Ready | E29 | M24 | All bugs fixed, all quality gates pass |

---

## Risk Register

| ID | Risk | Impact | Likelihood | Mitigation |
|----|------|--------|------------|------------|
| R12 | AWS/Azure sandbox credentials not available | E2E blocked indefinitely | High | Document required credentials and permissions. Provide setup guide. |
| R13 | Cloud SDK rate limits or quota during E2E | Tests fail intermittently | Low | Use dedicated test accounts with sufficient quotas. |

---

## Operating Procedure

### Definition of Done

A task is done when:
1. Code compiles with zero warnings.
2. All new code has unit tests with 100% coverage.
3. `go test ./... -race` passes with no regressions.
4. `golangci-lint run` passes with no new findings.
5. `gofmt -s` produces no changes.

### Commit Policy

- Always add tests when adding new implementation code.
- Always run relevant linters and formatters after code changes.
- Never commit files from different directories in the same commit.
- Conventional Commits: feat(deploy):, fix(deploy):, test(deploy):, docs:.

---

## Progress Log

### 2026-03-13 -- Plan Trimmed for Option 3

Trimmed completed epics E30-E40 (except T29.1-5) into docs/design.md. All Azure, managed hosting, registry, and hardening work is complete. Updated ADR 010 with official MCP registry relationship and Option 3 decision (keep registries separate). Only E2E validation tasks (T29.1-5) remain, blocked on real cloud sandbox credentials.

Changes:
- Trimmed 55 completed tasks (E30-E40) from plan. Knowledge preserved in docs/design.md (Azure deploy architecture, managed hosting, registry, production hardening, milestones M19-M23).
- Updated docs/adr/010-mcp-server-registry.md with "Relationship to the Official MCP Registry" section documenting Option 3 decision.
- Updated docs/design.md with Azure service mapping, managed hosting architecture, registry architecture, shared domain validation, production hardening details.

### 2026-03-13 -- Plan Created

Created Phase 2 plan covering four initiatives: D (Azure Container Apps), A (Managed Hosting), B (MCP Registry), C (E2E + Hardening). Defined 12 epics (E29-E40), 55 tasks. Trimmed completed AWS epics (E24-E28) from plan; knowledge preserved in docs/design.md (milestones M14-M18, AWS deploy architecture, deploy directory structure). Created three ADRs:
- docs/adr/008-azure-container-apps-deployment-target.md
- docs/adr/009-managed-mcp-hosting-platform.md
- docs/adr/010-mcp-server-registry.md

---

## Hand-off Notes

- All implementation is complete. Only E2E validation (T29.1-5) remains.
- E2E requires real AWS and Azure sandbox credentials. GCP E2E was already validated in M13.
- The mint registry (`sirerun/mcp-registry`) lists OpenAPI specs for generation. The official MCP registry (`registry.modelcontextprotocol.io`) lists pre-built servers. They are complementary. See docs/adr/010-mcp-server-registry.md.
- `mint registry install <name>` currently downloads the spec and prints instructions. A future enhancement could invoke `mint mcp generate` via subprocess.
- All tests pass with `-race`. golangci-lint is clean on all new code.
