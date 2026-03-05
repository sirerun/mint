# Overlay Example

This example demonstrates using OpenAPI Overlays to customize a spec for different environments.

## Files

- `spec.yaml` - Base OpenAPI spec
- `overlay.yaml` - Overlay document that modifies the base spec for production

## What the overlay does

1. Updates the API title to include "(Production)"
2. Updates the description for the production environment
3. Removes the `DELETE /items/{id}` operation (not exposed in production)

## Usage

```bash
# Apply the overlay
mint overlay apply spec.yaml overlay.yaml -o production-spec.yaml

# Generate an MCP server from the production spec
mint mcp generate --output ./server production-spec.yaml
```
