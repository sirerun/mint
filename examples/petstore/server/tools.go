package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleListPets(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	path := "/pets"

	var queryParts []string
	if v, ok := args["limit"]; ok {
		queryParts = append(queryParts, fmt.Sprintf("limit=%v", v))
	}

	if len(queryParts) > 0 {
		path = path + "?" + strings.Join(queryParts, "&")
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("list_pets: %w", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("list_pets: marshaling response: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleCreatePet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	path := "/pets"

	var queryParts []string

	if len(queryParts) > 0 {
		path = path + "?" + strings.Join(queryParts, "&")
	}

	bodyParams := make(map[string]interface{})
	if v, ok := args["name"]; ok {
		bodyParams["name"] = v
	}
	if v, ok := args["tag"]; ok {
		bodyParams["tag"] = v
	}

	resp, err := s.client.Do(ctx, "POST", path, bodyParams)
	if err != nil {
		return nil, fmt.Errorf("create_pet: %w", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("create_pet: marshaling response: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) handleShowPetById(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	path := "/pets/{petId}"
	if v, ok := args["petId"]; ok {
		path = strings.Replace(path, "{petId}", fmt.Sprintf("%v", v), 1)
	}

	var queryParts []string

	if len(queryParts) > 0 {
		path = path + "?" + strings.Join(queryParts, "&")
	}

	resp, err := s.client.Do(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("show_pet_by_id: %w", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("show_pet_by_id: marshaling response: %w", err)
	}

	return mcp.NewToolResultText(string(data)), nil
}

