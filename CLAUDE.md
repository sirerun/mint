# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in the mint repo.

The long-term ambition is $5B ARR by 2031, but the **operating target is a $150–400M ARR base case** ($1B held as a gated bull) won through **one wedge**: auditable, effectively-once, HITL-gated agents for money-touching regulated operations (EU AI Act forced-buy), where the decisive lever is **NRR via the value-fee mechanic, not raw logo count**. **If any effort derails this — especially breadth (10 product lines, marketplace-as-primary, speculative primitives, sovereign/VLA) — flag it and discard it.** Full plan + rationale: `docs/1b-arr-operating-plan.md` and ADR 184 (sirerun/docs).

## What this repo is

`mint` is a Go CLI and library that generates MCP (Model Context Protocol) servers from OpenAPI 3.x specs. It also provides a registry for discovering and publishing MCP servers, plus deployment helpers for managed/GCP/AWS hosting. Entry point: `cmd/mint/main.go`.

## Shared Docs Repo

Cross-repo planning and documentation lives in a dedicated git repo: `github.com/sirerun/docs`, checked out at `/Users/dndungu/Code/sirerun/docs/`. This is the single source of truth for `plan.md` (cross-repo execution plan used by /plan and /apply), `adr/` (cross-repo ADRs), `devlog.md` (investigations/benchmarks), `usecases.md`, `design.md`, and `content-classification.md`.

Work in this project is typically cross-repo. Always read/update the plan in the shared docs repo, not a per-repo copy. Commit docs changes via PR to `sirerun/docs` independently from code PRs.

## Staging Environment — HIBERNATED

`sire-staging.run` is temporarily hibernated (E3 in the shared docs/plan.md) to reduce cloud costs until funding closes. Do not deploy or test against staging. Tests target production using dedicated `qa+bot@sire.run` accounts in sandboxed workspaces. Hibernated (deleted): staging Cloud SQL, Redis, GKE. Preserved: secrets, Artifact Registry, DNS, KMS, IAM, Pulumi state. Revival: revert E3 gates and `pulumi up --stack staging`.

## No Manual DevOps — IaC + Release Pipeline Only

Production and staging are managed exclusively through IaC and the CI/CD release pipeline. Banned: `kubectl set/edit/scale/patch/delete` and `kubectl apply` against staging/prod, `gcloud secrets create/add/delete` and other imperative `gcloud` mutations, direct prod DB writes, hot-patching pods, re-tagging or force-pushing. Required path: edit IaC → PR → CI → rebase merge → tag release → deploy workflow → verify via workflow checks. Read-only diagnostics (`kubectl get/describe/logs`, `gcloud ... list/access`, `gh run view`) are fine. Agents: never run mutating commands against live infra; open a PR.

## Commands

```bash
go test ./...
go build ./cmd/mint
golangci-lint run --timeout=10m
```

## House style

- Prefer Go standard library over third-party packages.
- Use table-driven tests with the standard `testing` package (no testify).
- Conventional Commits for commit messages.
