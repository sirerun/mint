# Mint Deploy -- Production-Ready GCP Deployment

## Context

See docs/design.md for full project context, architecture, and conventions.

All deploy commands are fully implemented and validated end-to-end against a real GCP project (sire-sandbox).

---

## Checkable Work Breakdown

### Epic E23: Validation and Cleanup

- [x] T23.3 End-to-end validation with Twitter API v2 spec  Owner: TBD  Est: 1h  Completed: 2026-03-12
  - Generated MCP server from `https://api.twitter.com/2/openapi.json` (156 tools).
  - Built container and deployed to Cloud Run in sire-sandbox.
  - `mint deploy status` showed service info and revisions.
  - `curl /health` returned HTTP 200 (with auth).
  - `mint deploy rollback` shifted traffic from revision 00002 to 00001.
  - Three bugs found and fixed during validation:
    - Cloud Build source bucket not auto-created for new projects.
    - Empty ServiceName produced malformed image URI.
    - Container defaulted to stdio transport; revision names were full resource paths.

---

## Timeline and Milestones

| Milestone | ID | Dependencies | Exit Criteria |
|-----------|----|--------------|---------------|
| M13: Production Ready | E23 | M11, M12 (complete) | Manual e2e validation passes with Twitter API v2 spec. COMPLETE. |

---

## Progress Log

### 2026-03-12 -- E2E Validation Complete

Completed T23.3 against sire-sandbox with Twitter API v2 spec. Found and fixed three bugs:
1. Cloud Build adapter did not auto-create the source bucket (57619df).
2. DeployConfig.Validate did not derive ServiceName or default ImageTag (ecf6165).
3. Cloud Run adapter did not pass --transport sse args; revision names were full resource paths breaking rollback (57031c8).
All deploy commands verified: deploy gcp, deploy status, deploy rollback. Milestone M13 achieved.

### 2026-03-12 -- Updated Validation Target

Updated T23.3 to use Twitter API v2 OpenAPI spec instead of petstore example.

### 2026-03-12 -- Trimmed Plan

Trimmed plan. Stable knowledge preserved in docs/design.md and docs/adr/.
