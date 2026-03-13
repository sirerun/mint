package mcpgen

import (
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	server, err := ParseFile("../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("ParseFile() error: %v", err)
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

	toolNames := make(map[string]bool)
	for _, tool := range server.Tools {
		toolNames[tool.Name] = true
	}

	for _, want := range []string{"list_pets", "create_pet", "show_pet_by_id"} {
		if !toolNames[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestParseReader(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: "Test API"
  version: "0.1.0"
paths:
  /health:
    get:
      operationId: checkHealth
      summary: "Health check"
      responses:
        "200":
          description: OK
`
	server, err := ParseReader(strings.NewReader(spec), "inline")
	if err != nil {
		t.Fatalf("ParseReader() error: %v", err)
	}

	if server.Name != "test-api" {
		t.Errorf("Name = %q, want %q", server.Name, "test-api")
	}

	if len(server.Tools) != 1 {
		t.Fatalf("len(Tools) = %d, want 1", len(server.Tools))
	}

	if server.Tools[0].Name != "check_health" {
		t.Errorf("tool name = %q, want %q", server.Tools[0].Name, "check_health")
	}

	if server.Tools[0].HTTPMethod != "GET" {
		t.Errorf("tool method = %q, want GET", server.Tools[0].HTTPMethod)
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
