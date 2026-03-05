# Quality Gates

## Go Quality Profile

| Gate | Command | Status |
|------|---------|--------|
| Build | `go build ./...` | PASS |
| Test | `go test ./... -race -timeout 120s` | PASS |
| Vet | `go vet ./...` | PASS |
| Lint | `golangci-lint run ./...` | PASS (0 issues) |

## Test Coverage

| Package | Coverage |
|---------|----------|
| internal/loader | 82.6% |
| internal/diff | 84.8% |
| internal/merge | 88.2% |
| internal/mcpgen | 65.3% |
| internal/mcpgen/golang | 82.1% |
| internal/validate | 74.0% |
| internal/lint | 78.5% |
| internal/color | 90.0% |
| internal/overlay | 80.0% |
| internal/transform | 76.3% |
| cmd/mint | 34.6% |

## Dependencies

| Dependency | License | Purpose |
|-----------|---------|---------|
| pb33f/libopenapi | BSD-3 | OpenAPI parsing |
| pb33f/ordered-map/v2 | BSD-3 | Ordered map (transitive) |
| go.yaml.in/yaml/v4 | MIT | YAML serialization (transitive) |
| golang.org/x/term | BSD-3 | Terminal detection for colored output |

### Runtime Dependencies (in generated servers)

| Dependency | License | Purpose |
|-----------|---------|---------|
| mark3labs/mcp-go | MIT | MCP SDK for generated Go servers |

## Pre-commit Hooks

- golangci-lint runs on staged packages
- `go test ./...` runs on every commit
