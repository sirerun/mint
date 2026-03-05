# Contributing to mint

Thank you for your interest in contributing to mint.

## Development Setup

```bash
git clone https://github.com/sirerun/mint.git
cd mint
go build ./...
go test ./...
```

## Requirements

- Go 1.23+
- golangci-lint v2+

## Making Changes

1. Fork and clone the repository
2. Create a branch for your changes
3. Write tests for new functionality
4. Ensure `go test ./...` passes
5. Ensure `golangci-lint run ./...` passes
6. Submit a pull request

## Code Style

- Follow standard Go conventions
- Use the standard library where possible
- Write table-driven tests
- Use the `testing` package (no testify)

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(cli): add new command
fix(loader): handle edge case
docs(readme): update install instructions
```

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
