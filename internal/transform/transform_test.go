package transform

import (
	"strings"
	"testing"
)

func TestFilterByExcludePaths(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /users:
    get:
      summary: List users
  /internal/metrics:
    get:
      summary: Metrics
  /pets:
    get:
      summary: List pets
`)
	result, err := FilterOperations(spec, nil, []string{"/internal/*"})
	if err != nil {
		t.Fatalf("FilterOperations() error: %v", err)
	}
	if strings.Contains(string(result), "metrics") {
		t.Error("expected /internal/metrics to be excluded")
	}
	if !strings.Contains(string(result), "/users") {
		t.Error("expected /users to be included")
	}
}

func TestFilterByTags(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /users:
    get:
      summary: List users
      tags:
        - users
  /pets:
    get:
      summary: List pets
      tags:
        - pets
`)
	result, err := FilterOperations(spec, []string{"users"}, nil)
	if err != nil {
		t.Fatalf("FilterOperations() error: %v", err)
	}
	if !strings.Contains(string(result), "/users") {
		t.Error("expected /users to be included")
	}
	if strings.Contains(string(result), "/pets") {
		t.Error("expected /pets to be excluded")
	}
}

func TestNormalize(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
paths:
  /pets:
    get:
      summary: List
info:
  version: "1.0"
  title: Test
`)
	result, err := Normalize(spec)
	if err != nil {
		t.Fatalf("Normalize() error: %v", err)
	}

	// Run again to verify idempotency
	result2, err := Normalize(result)
	if err != nil {
		t.Fatalf("Normalize() second pass error: %v", err)
	}
	if string(result) != string(result2) {
		t.Error("Normalize is not idempotent")
	}
}

func TestRemoveUnusedComponents(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
    Unused:
      type: object
      properties:
        foo:
          type: string
`)
	result, err := RemoveUnusedComponents(spec)
	if err != nil {
		t.Fatalf("RemoveUnusedComponents() error: %v", err)
	}
	if !strings.Contains(string(result), "Pet") {
		t.Error("expected Pet schema to be kept")
	}
	if strings.Contains(string(result), "Unused") {
		t.Error("expected Unused schema to be removed")
	}
}
