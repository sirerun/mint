package mcpgen

import "testing"

func TestRenameTools(t *testing.T) {
	tools := []MCPTool{
		{Name: "list_pets", Description: "List pets"},
		{Name: "create_pet", Description: "Create a pet"},
		{Name: "show_pet_by_id", Description: "Show pet"},
	}

	mapping := map[string]string{
		"list_pets":  "get_all_pets",
		"create_pet": "add_pet",
	}

	result := RenameTools(tools, mapping)

	if result[0].Name != "get_all_pets" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "get_all_pets")
	}
	if result[1].Name != "add_pet" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "add_pet")
	}
	if result[2].Name != "show_pet_by_id" {
		t.Errorf("result[2].Name = %q, want %q (should be unchanged)", result[2].Name, "show_pet_by_id")
	}
}

func TestRenameToolsEmpty(t *testing.T) {
	tools := []MCPTool{{Name: "test"}}
	result := RenameTools(tools, nil)
	if len(result) != 1 || result[0].Name != "test" {
		t.Error("nil mapping should return tools unchanged")
	}
}

func TestRenameToolsNoMatch(t *testing.T) {
	tools := []MCPTool{{Name: "test"}}
	result := RenameTools(tools, map[string]string{"other": "renamed"})
	if result[0].Name != "test" {
		t.Errorf("non-matching mapping should leave name as %q", "test")
	}
}
