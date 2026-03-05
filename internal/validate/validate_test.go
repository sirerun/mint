package validate

import (
	"os"
	"testing"
)

func TestValidSpec(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	result := Spec(data, "petstore.yaml")
	if !result.Valid {
		t.Errorf("expected valid spec, got invalid")
		for _, d := range result.Diagnostics {
			t.Logf("  %s", d)
		}
	}
}

func TestInvalidSpec(t *testing.T) {
	data, err := os.ReadFile("../../testdata/invalid.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	result := Spec(data, "invalid.yaml")
	if result.Valid {
		t.Error("expected invalid spec, got valid")
	}
	if len(result.Diagnostics) == 0 {
		t.Error("expected diagnostics for invalid spec")
	}
}

func TestEmptyPaths(t *testing.T) {
	data := []byte(`openapi: "3.0.3"
info:
  title: Empty
  version: "1.0"
paths: {}
`)
	result := Spec(data, "empty.yaml")
	// Should be valid but have a warning about no paths
	hasPathsWarning := false
	for _, d := range result.Diagnostics {
		if d.RuleID == "paths-defined" {
			hasPathsWarning = true
		}
	}
	if !hasPathsWarning {
		t.Error("expected warning about no paths defined")
	}
}

func TestMissingOperationID(t *testing.T) {
	data := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: OK
`)
	result := Spec(data, "test.yaml")
	hasOpIDWarning := false
	for _, d := range result.Diagnostics {
		if d.RuleID == "operation-id" {
			hasOpIDWarning = true
		}
	}
	if !hasOpIDWarning {
		t.Error("expected warning about missing operationId")
	}
}

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		d    Diagnostic
		want string
	}{
		{
			Diagnostic{Severity: "error", Message: "bad", Path: "spec.yaml"},
			"[error] spec.yaml: bad",
		},
		{
			Diagnostic{Severity: "warning", Message: "meh"},
			"[warning] meh",
		},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}
