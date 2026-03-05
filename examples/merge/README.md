# Merge Example

This example demonstrates merging two independent OpenAPI specs into one combined spec.

## Files

- `users-api.yaml` - Users microservice API
- `products-api.yaml` - Products microservice API

## Usage

```bash
# Merge the two specs
mint merge users-api.yaml products-api.yaml -o merged.yaml

# The merged spec will contain all paths and schemas from both APIs.
# You can then generate an MCP server from the merged spec:
mint mcp generate --output ./server merged.yaml
```

## Conflict Strategies

If two specs have overlapping paths, you can control the behavior:

```bash
# Fail on conflicts (default)
mint merge --on-conflict fail a.yaml b.yaml

# Skip conflicting paths from the second spec
mint merge --on-conflict skip a.yaml b.yaml

# Rename conflicting paths
mint merge --on-conflict rename a.yaml b.yaml
```
