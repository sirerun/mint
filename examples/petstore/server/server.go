package main

import (
	"context"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server and HTTP client.
type Server struct {
	mcpServer *server.MCPServer
	client    *APIClient
}

// NewServer creates a new MCP server with all tools registered.
func NewServer(baseURL, apiKey string) *Server {
	s := &Server{
		client: NewAPIClient(baseURL, apiKey),
	}

	mcpSrv := server.NewMCPServer(
		"petstore",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	mcpSrv.AddTool(
		mcp.Tool{
			Name:        "list_pets",
			Description: "List all pets",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"limit": map[string]interface{}{
						"type":   "integer",
						"format": "int32",
					},
				},
			},
		},
		s.handleListPets,
	)

	mcpSrv.AddTool(
		mcp.Tool{
			Name:        "create_pet",
			Description: "Create a pet",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
					"tag": map[string]interface{}{
						"type": "string",
					},
				},
				Required: []string{"name"},
			},
		},
		s.handleCreatePet,
	)

	mcpSrv.AddTool(
		mcp.Tool{
			Name:        "show_pet_by_id",
			Description: "Info for a specific pet",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"petId": map[string]interface{}{
						"type": "string",
					},
				},
				Required: []string{"petId"},
			},
		},
		s.handleShowPetById,
	)

	s.mcpServer = mcpSrv
	return s
}

// ServeStdio starts the server using stdio transport.
func (s *Server) ServeStdio() error {
	srv := server.NewStdioServer(s.mcpServer)
	return srv.Listen(context.Background(), os.Stdin, os.Stdout)
}

// ServeSSE starts the server using SSE transport.
func (s *Server) ServeSSE(addr string) error {
	srv := server.NewSSEServer(s.mcpServer)
	return srv.Start(addr)
}
