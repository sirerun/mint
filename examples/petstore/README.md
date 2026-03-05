# Petstore MCP Server Example

This example demonstrates generating an MCP server from the classic Petstore OpenAPI spec.

## Files

- `petstore.yaml` - The OpenAPI 3.0 spec
- `server/` - The generated Go MCP server

## Generate

```bash
mint mcp generate --output ./server petstore.yaml
```

## Build and Run

```bash
cd server
go mod tidy
go build -o petstore .
./petstore --transport stdio
```

## Tools

The generated server exposes 3 MCP tools:

| Tool | Description | Method | Path |
|------|-------------|--------|------|
| `list_pets` | List all pets | GET | /pets |
| `create_pet` | Create a pet | POST | /pets |
| `show_pet_by_id` | Info for a specific pet | GET | /pets/{petId} |

## Claude Desktop Configuration

```json
{
  "mcpServers": {
    "petstore": {
      "command": "/path/to/petstore",
      "args": ["--transport", "stdio", "--base-url", "https://petstore.swagger.io/v2"]
    }
  }
}
```
