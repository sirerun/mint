package transform

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestConvertSwagger(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore-swagger2.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	result, err := ConvertSwagger(data)
	if err != nil {
		t.Fatalf("ConvertSwagger() error: %v", err)
	}

	var spec map[string]interface{}
	if err := yaml.Unmarshal(result, &spec); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	// Check openapi version
	if v, _ := spec["openapi"].(string); v != "3.0.3" {
		t.Errorf("openapi = %q, want %q", v, "3.0.3")
	}

	// Check swagger key is gone
	if _, ok := spec["swagger"]; ok {
		t.Error("converted spec should not have 'swagger' key")
	}

	// Check servers
	servers, ok := spec["servers"].([]interface{})
	if !ok || len(servers) == 0 {
		t.Fatal("expected servers to be populated")
	}
	s0, _ := servers[0].(map[string]interface{})
	url, _ := s0["url"].(string)
	if url != "https://petstore.example.com/v1" {
		t.Errorf("server url = %q, want %q", url, "https://petstore.example.com/v1")
	}

	// Check paths exist
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("expected paths")
	}
	if _, ok := paths["/pets"]; !ok {
		t.Error("expected /pets path")
	}
	if _, ok := paths["/pets/{petId}"]; !ok {
		t.Error("expected /pets/{petId} path")
	}

	// Check components/schemas (converted from definitions)
	components, ok := spec["components"].(map[string]interface{})
	if !ok {
		t.Fatal("expected components")
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		t.Fatal("expected components/schemas")
	}
	if _, ok := schemas["Pet"]; !ok {
		t.Error("expected Pet schema in components")
	}

	// Check $ref conversion
	petsPath, _ := paths["/pets"].(map[string]interface{})
	getOp, _ := petsPath["get"].(map[string]interface{})
	responses, _ := getOp["responses"].(map[string]interface{})
	resp200, _ := responses["200"].(map[string]interface{})
	content, _ := resp200["content"].(map[string]interface{})
	jsonContent, _ := content["application/json"].(map[string]interface{})
	schema, _ := jsonContent["schema"].(map[string]interface{})
	items, _ := schema["items"].(map[string]interface{})
	ref, _ := items["$ref"].(string)
	if ref != "#/components/schemas/Pet" {
		t.Errorf("$ref = %q, want %q", ref, "#/components/schemas/Pet")
	}

	// Check requestBody for POST
	postOp, _ := petsPath["post"].(map[string]interface{})
	reqBody, ok := postOp["requestBody"].(map[string]interface{})
	if !ok {
		t.Error("expected requestBody for POST /pets")
	} else {
		rbContent, _ := reqBody["content"].(map[string]interface{})
		if _, ok := rbContent["application/json"]; !ok {
			t.Error("expected application/json in requestBody content")
		}
	}

	// Check security schemes
	secSchemes, ok := components["securitySchemes"].(map[string]interface{})
	if !ok {
		t.Fatal("expected securitySchemes")
	}
	apiKey, ok := secSchemes["api_key"].(map[string]interface{})
	if !ok {
		t.Fatal("expected api_key security scheme")
	}
	if typ, _ := apiKey["type"].(string); typ != "apiKey" {
		t.Errorf("api_key type = %q, want %q", typ, "apiKey")
	}
}

func TestConvertSwaggerNotSwagger(t *testing.T) {
	data := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
`)
	_, err := ConvertSwagger(data)
	if err == nil {
		t.Error("expected error for non-Swagger spec")
	}
}

func TestConvertSwaggerBasicAuth(t *testing.T) {
	data := []byte(`swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths: {}
securityDefinitions:
  basic:
    type: basic
`)
	result, err := ConvertSwagger(data)
	if err != nil {
		t.Fatalf("ConvertSwagger() error: %v", err)
	}

	var spec map[string]interface{}
	if err := yaml.Unmarshal(result, &spec); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	components, _ := spec["components"].(map[string]interface{})
	schemes, _ := components["securitySchemes"].(map[string]interface{})
	basic, _ := schemes["basic"].(map[string]interface{})
	if typ, _ := basic["type"].(string); typ != "http" {
		t.Errorf("basic type = %q, want %q", typ, "http")
	}
	if scheme, _ := basic["scheme"].(string); scheme != "basic" {
		t.Errorf("basic scheme = %q, want %q", scheme, "basic")
	}
}

func TestConvertSwaggerOAuth2(t *testing.T) {
	data := []byte(`swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths: {}
securityDefinitions:
  oauth:
    type: oauth2
    flow: accessCode
    authorizationUrl: https://example.com/auth
    tokenUrl: https://example.com/token
    scopes:
      read: Read access
`)
	result, err := ConvertSwagger(data)
	if err != nil {
		t.Fatalf("ConvertSwagger() error: %v", err)
	}

	var spec map[string]interface{}
	if err := yaml.Unmarshal(result, &spec); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	components, _ := spec["components"].(map[string]interface{})
	schemes, _ := components["securitySchemes"].(map[string]interface{})
	oauth, _ := schemes["oauth"].(map[string]interface{})
	if typ, _ := oauth["type"].(string); typ != "oauth2" {
		t.Errorf("oauth type = %q, want %q", typ, "oauth2")
	}
	flows, _ := oauth["flows"].(map[string]interface{})
	if _, ok := flows["authorizationCode"]; !ok {
		t.Error("expected authorizationCode flow (converted from accessCode)")
	}
}

func TestConvertSwaggerNoHost(t *testing.T) {
	data := []byte(`swagger: "2.0"
info:
  title: Test
  version: "1.0"
paths: {}
`)
	result, err := ConvertSwagger(data)
	if err != nil {
		t.Fatalf("ConvertSwagger() error: %v", err)
	}

	var spec map[string]interface{}
	if err := yaml.Unmarshal(result, &spec); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}

	if _, ok := spec["servers"]; ok {
		t.Error("expected no servers when no host defined")
	}
}
