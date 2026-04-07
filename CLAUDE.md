# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in the mint repo.

The most important goal is that we need to make Sire have $5B ARR by end of 2031, if any effort derails us, let me know and discard it.

## What this repo is

`mint` is a Go CLI and library that generates MCP (Model Context Protocol) servers from OpenAPI 3.x specs. It also provides a registry for discovering and publishing MCP servers, plus deployment helpers for managed/GCP/AWS hosting. Entry point: `cmd/mint/main.go`.

## Shared Docs Repo

Cross-repo planning and documentation lives in a dedicated git repo: `github.com/sirerun/docs`, checked out at `/Users/dndungu/Code/sirerun/docs/`. This is the single source of truth for `plan.md` (cross-repo execution plan used by /plan and /apply), `adr/` (cross-repo ADRs), `devlog.md` (investigations/benchmarks), `usecases.md`, `design.md`, and `content-classification.md`.

Work in this project is typically cross-repo. Always read/update the plan in the shared docs repo, not a per-repo copy. Commit docs changes via PR to `sirerun/docs` independently from code PRs.

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
