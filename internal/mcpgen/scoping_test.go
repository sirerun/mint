package mcpgen

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/libopenapi"
)

// buildDoc parses an inline OpenAPI spec into the v3 model Convert consumes.
func buildScopingDoc(t *testing.T, spec string) *MCPServer {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(spec))
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	model, buildErr := doc.BuildV3Model()
	if buildErr != nil {
		t.Fatalf("BuildV3Model: %v", buildErr)
	}
	server, err := Convert(&model.Model)
	if err != nil {
		t.Fatalf("Convert: %v", err)
	}
	return server
}

// TestConvertScopingExtension_Present verifies the top-level `x-sire-scoping`
// extension is captured as canonical JSON on MCPServer.Scoping.
func TestConvertScopingExtension_Present(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: shopify
  version: 1.0.0
x-sire-scoping:
  pattern: tenant_scoped
  scopeParam: customer_id
  derivation: external_principal
paths: {}
`
	server := buildScopingDoc(t, spec)
	if len(server.Scoping) == 0 {
		t.Fatal("Scoping is empty; want the x-sire-scoping extension")
	}

	var got map[string]any
	if err := json.Unmarshal(server.Scoping, &got); err != nil {
		t.Fatalf("Scoping is not valid JSON: %v (%s)", err, server.Scoping)
	}
	want := map[string]any{
		"pattern":    "tenant_scoped",
		"scopeParam": "customer_id",
		"derivation": "external_principal",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("Scoping[%q] = %v, want %v", k, got[k], v)
		}
	}
}

// TestConvertScopingExtension_Absent verifies a spec without the extension
// yields a nil Scoping (omitted from JSON), the default-deny posture.
func TestConvertScopingExtension_Absent(t *testing.T) {
	spec := `openapi: 3.0.0
info:
  title: noscope
  version: 1.0.0
paths: {}
`
	server := buildScopingDoc(t, spec)
	if server.Scoping != nil {
		t.Errorf("Scoping = %s, want nil for a spec with no x-sire-scoping", server.Scoping)
	}

	// And it must be omitted from the marshaled server.
	out, err := json.Marshal(server)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if _, present := m["scoping"]; present {
		t.Error("scoping key present in JSON; want omitted when absent")
	}
}
