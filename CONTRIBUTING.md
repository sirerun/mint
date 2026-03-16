# Contributing to mint

Thank you for your interest in contributing to mint. This guide covers development setup, architecture, coding standards, and the pull request process.

## Development Setup

```bash
git clone https://github.com/sirerun/mint.git
cd mint
go build ./...
go test ./... -race
```

### Prerequisites

- Go 1.25+
- golangci-lint v2+ (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)

### Quality Gates

Every change must pass all four gates before merge:

```bash
go build ./...                          # Build
go test ./... -race -timeout 120s       # Test (with race detector)
go vet ./...                            # Vet
golangci-lint run ./...                 # Lint (zero findings required)
```

CI runs these automatically on every pull request.

## Project Structure

```
cmd/mint/           CLI entry point and subcommand wiring
internal/
  generate/         OpenAPI → MCP server code generation
  deploy/           Deployment orchestration
    gcp/            Google Cloud Run deployment
    aws/            AWS ECS Fargate deployment
  openapi/          OpenAPI parsing, validation, linting, diffing
  mcp/              MCP protocol types and server runtime
docs/
  plan.md           Development plan and task breakdown
  adr/              Architecture Decision Records
```

### Deploy Architecture

The deploy packages use a three-layer adapter pattern:

```
Deployer (orchestrator)
  → Bridge Adapters (business logic: ensure-if-missing, retry, poll)
    → SDK Adapters (thin wrappers around cloud SDK calls)
```

Each layer is separated by Go interfaces, making everything testable with mocks. The SDK adapters extract narrow interfaces for the underlying cloud SDK methods so unit tests never call real cloud APIs.

## Making Changes

1. Fork and clone the repository.
2. Create a feature branch from `main`.
3. Write tests first, then implement (TDD preferred).
4. Ensure all quality gates pass locally.
5. Submit a pull request against `main`.

### Branch Naming

Use descriptive branch names:

```
feat/aws-deploy-canary
fix/openapi-nullable-handling
docs/update-deploy-readme
```

## Code Style

### Go Conventions

- Prefer the Go standard library over third-party packages.
- Use interface segregation -- define narrow interfaces where they're consumed.
- Apply single responsibility. Avoid catch-all utility packages.
- Keep exported APIs minimal. Only export what other packages need.
- Use `context.Context` as the first parameter for functions that do I/O.

### Testing

- Write table-driven tests using the `testing` package.
- Do **not** add testify or other test assertion libraries.
- Use mock interfaces for external dependencies -- never call real cloud APIs in unit tests.
- Target 100% coverage for new packages, 90%+ minimum for changes to existing packages.
- Run tests with `-race` to catch data races.

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {name: "valid input", input: "hello", want: "HELLO"},
        {name: "empty input", input: "", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr = %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Error Handling

- Use sentinel errors (`var ErrNotFound = errors.New(...)`) for expected conditions.
- Wrap errors with context: `fmt.Errorf("describing service %s: %w", name, err)`.
- Check error return values from all function calls (enforced by `errcheck` linter).

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/). Releases are automated via [go-semantic-release](https://github.com/go-semantic-release/semantic-release) -- commit message prefixes determine the version bump:

| Prefix | Version Bump | Example |
|--------|-------------|---------|
| `feat:` | Minor (0.x.0) | `feat(deploy): add AWS canary traffic splitting` |
| `fix:` | Patch (0.0.x) | `fix(loader): handle nullable schema refs` |
| `feat!:` or `BREAKING CHANGE:` | Major (x.0.0) | `feat!: remove deprecated --format flag` |
| `docs:`, `test:`, `chore:` | No release | `docs: update deploy examples` |

Scope is optional but encouraged. Use the package or feature area: `deploy`, `cli`, `openapi`, `generate`, `ci`.

```
feat(deploy): add AWS ECS Fargate deployment target
fix(openapi): resolve circular $ref in schema validation
test(deploy): achieve 100% coverage for AWS deploy package
docs(readme): add AWS deploy examples
chore(deps): update mcp-go to v0.12
```

## Pull Requests

### PR Requirements

- All quality gates pass (CI enforces this).
- New code has tests.
- No secrets, credentials, or `.env` files committed.
- PR description explains **what** changed and **why**.

### PR Template

```markdown
## Summary
- Brief description of the change

## Test Plan
- [ ] Unit tests added/updated
- [ ] `go test ./... -race` passes
- [ ] `golangci-lint run ./...` clean
```

### Review Process

- PRs are rebased and merged (no squash, no merge commits).
- At least one approval required before merge.
- Address review feedback with new commits (don't force-push during review).

## Adding a New Deploy Provider

To add a new cloud provider (e.g., Azure):

1. Create `internal/deploy/azure/` with the three-layer adapter pattern.
2. Define orchestrator interfaces in `deploy.go` (e.g., `RegistryProvisioner`, `ServiceDeployer`).
3. Implement SDK adapters with extracted SDK interfaces for testability.
4. Implement bridge adapters connecting SDK adapters to orchestrator interfaces.
5. Wire the CLI in `cmd/mint/deploy.go` with a new `case "azure":` block.
6. Add `--provider azure` support to `status` and `rollback` commands.
7. Add workflow generation in `workflow.go` and OIDC setup if applicable.
8. Achieve 100% test coverage.
9. Update README.md with usage examples.

## Architecture Decision Records

Significant design decisions are recorded in `docs/adr/`. When proposing a structural change, create a new ADR:

```
docs/adr/YYYYMMDD-short-description.md
```

Include: context, decision, consequences, and alternatives considered.

## Getting Help

- Open an issue at [github.com/sirerun/mint/issues](https://github.com/sirerun/mint/issues)
- Check existing ADRs in `docs/adr/` for design context

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
