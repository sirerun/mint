// Package mcpgen provides a public API for parsing OpenAPI specs into MCP
// server models. It wraps the internal converter and loader packages.
//
// Usage:
//
//	server, err := mcpgen.ParseFile("petstore.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, tool := range server.Tools {
//	    fmt.Println(tool.Name, tool.HTTPMethod, tool.HTTPPath)
//	}
package mcpgen

import (
	"io"

	"github.com/sirerun/mint/internal/loader"
	internal "github.com/sirerun/mint/internal/mcpgen"
)

// MCPServer represents a parsed MCP server with its tools and configuration.
type MCPServer = internal.MCPServer

// MCPTool represents a single MCP tool derived from an OpenAPI operation.
type MCPTool = internal.MCPTool

// MCPToolParam describes a single parameter for an MCP tool.
type MCPToolParam = internal.MCPToolParam

// MCPAuth describes authentication configuration for the server.
type MCPAuth = internal.MCPAuth

// JSONSchema is a simplified JSON Schema representation for MCP tool inputs.
type JSONSchema = internal.JSONSchema

// ParseFile reads an OpenAPI spec from a file path or URL and returns an
// MCPServer model. Supports OpenAPI 3.0 and 3.1 in YAML or JSON format.
func ParseFile(source string) (*MCPServer, error) {
	result, err := loader.Load(source)
	if err != nil {
		return nil, err
	}
	return internal.Convert(result.Model)
}

// ParseReader reads an OpenAPI spec from an io.Reader and returns an
// MCPServer model. The sourcePath is used for error messages only.
func ParseReader(r io.Reader, sourcePath string) (*MCPServer, error) {
	result, err := loader.LoadReader(r, sourcePath)
	if err != nil {
		return nil, err
	}
	return internal.Convert(result.Model)
}
