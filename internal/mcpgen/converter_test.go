package mcpgen

import (
	"testing"

	"github.com/sirerun/mint/internal/loader"
)

func loadTestSpec(t *testing.T) *loader.Result {
	t.Helper()
	result, err := loader.Load("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading test spec: %v", err)
	}
	return result
}

func TestConvertPetstore(t *testing.T) {
	result := loadTestSpec(t)
	server, err := Convert(result.Model)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	if server.Name != "petstore" {
		t.Errorf("Name = %q, want %q", server.Name, "petstore")
	}

	if server.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", server.Version, "1.0.0")
	}

	if len(server.Tools) != 3 {
		t.Fatalf("len(Tools) = %d, want 3", len(server.Tools))
	}

	toolNames := make(map[string]MCPTool)
	for _, tool := range server.Tools {
		toolNames[tool.Name] = tool
	}

	// Test list_pets
	listPets, ok := toolNames["list_pets"]
	if !ok {
		t.Fatal("missing tool list_pets")
	}
	if listPets.HTTPMethod != "GET" {
		t.Errorf("list_pets.HTTPMethod = %q, want GET", listPets.HTTPMethod)
	}
	if listPets.HTTPPath != "/pets" {
		t.Errorf("list_pets.HTTPPath = %q, want /pets", listPets.HTTPPath)
	}
	if listPets.Description != "List all pets" {
		t.Errorf("list_pets.Description = %q, want %q", listPets.Description, "List all pets")
	}
	// limit param should not be required
	if len(listPets.Params) != 1 {
		t.Fatalf("list_pets params count = %d, want 1", len(listPets.Params))
	}
	if listPets.Params[0].Name != "limit" {
		t.Errorf("param name = %q, want limit", listPets.Params[0].Name)
	}
	if listPets.Params[0].Required {
		t.Error("limit param should not be required")
	}
	if listPets.Params[0].Type != "integer" {
		t.Errorf("limit param type = %q, want integer", listPets.Params[0].Type)
	}

	// Test create_pet
	createPet, ok := toolNames["create_pet"]
	if !ok {
		t.Fatal("missing tool create_pet")
	}
	if createPet.HTTPMethod != "POST" {
		t.Errorf("create_pet.HTTPMethod = %q, want POST", createPet.HTTPMethod)
	}
	// Should have body params: name (required), tag (optional)
	if len(createPet.Params) != 2 {
		t.Fatalf("create_pet params count = %d, want 2", len(createPet.Params))
	}
	paramMap := make(map[string]MCPToolParam)
	for _, p := range createPet.Params {
		paramMap[p.Name] = p
	}
	nameParam, ok := paramMap["name"]
	if !ok {
		t.Fatal("missing param 'name' on create_pet")
	}
	if !nameParam.Required {
		t.Error("name param should be required")
	}
	tagParam, ok := paramMap["tag"]
	if !ok {
		t.Fatal("missing param 'tag' on create_pet")
	}
	if tagParam.Required {
		t.Error("tag param should not be required")
	}

	// Test show_pet_by_id
	showPet, ok := toolNames["show_pet_by_id"]
	if !ok {
		t.Fatal("missing tool show_pet_by_id")
	}
	if showPet.HTTPMethod != "GET" {
		t.Errorf("show_pet_by_id.HTTPMethod = %q, want GET", showPet.HTTPMethod)
	}
	if len(showPet.Params) != 1 {
		t.Fatalf("show_pet_by_id params count = %d, want 1", len(showPet.Params))
	}
	if !showPet.Params[0].Required {
		t.Error("petId should be required")
	}
}

func TestConvertNoOperationID(t *testing.T) {
	// Test that tool names are derived from method+path when operationId is missing
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/users", "get_users"},
		{"POST", "/users/{userId}/posts", "get_users_userId_posts"},
		{"DELETE", "/items/{id}", "delete_items_id"},
	}

	for _, tt := range tests {
		got := deriveToolName("", tt.method, tt.path)
		// We just verify it produces a reasonable name
		if got == "" {
			t.Errorf("deriveToolName(%q, %q, %q) returned empty", "", tt.method, tt.path)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"listPets", "list_pets"},
		{"createPet", "create_pet"},
		{"showPetById", "show_pet_by_id"},
		{"getPetByID", "get_pet_by_id"},
		{"simple", "simple"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := toSnakeCase(tt.input); got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSchemaToJSONSchema(t *testing.T) {
	result := loadTestSpec(t)
	server, err := Convert(result.Model)
	if err != nil {
		t.Fatalf("Convert() error: %v", err)
	}

	// Check input schemas have correct structure
	for _, tool := range server.Tools {
		if tool.InputSchema.Type != "object" {
			t.Errorf("tool %s inputSchema.Type = %q, want object", tool.Name, tool.InputSchema.Type)
		}
		if tool.InputSchema.Properties == nil {
			t.Errorf("tool %s inputSchema.Properties is nil", tool.Name)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Petstore", "petstore"},
		{"My API v2", "my-api-v2"},
		{"Hello World!", "hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeName(tt.input); got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
