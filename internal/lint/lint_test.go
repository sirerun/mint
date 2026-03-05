package lint

import (
	"os"
	"testing"
)

func TestRunWithRecommendedRuleset(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	rs, _ := GetRuleset("recommended")
	result, err := Run(data, "petstore.yaml", rs)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if result.Errors > 0 {
		t.Errorf("expected no errors for petstore, got %d", result.Errors)
		for _, d := range result.Items {
			if d.Severity == SeverityError {
				t.Logf("  %s", d)
			}
		}
	}
}

func TestRunWithStrictRuleset(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	rs, _ := GetRuleset("strict")
	result, err := Run(data, "petstore.yaml", rs)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	// Strict ruleset should produce more findings than recommended
	if len(result.Items) == 0 {
		t.Error("expected some diagnostics with strict ruleset")
	}
}

func TestRunWithMinimalRuleset(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	rs, _ := GetRuleset("minimal")
	result, err := Run(data, "petstore.yaml", rs)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	// Minimal should have fewer findings than recommended
	rsRec, _ := GetRuleset("recommended")
	recResult, _ := Run(data, "petstore.yaml", rsRec)
	if len(result.Items) > len(recResult.Items) {
		t.Errorf("minimal ruleset produced more diagnostics (%d) than recommended (%d)",
			len(result.Items), len(recResult.Items))
	}
}

func TestRunMissingOperationID(t *testing.T) {
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
	rs, _ := GetRuleset("recommended")
	result, err := Run(data, "test.yaml", rs)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	hasOpIDWarning := false
	for _, d := range result.Items {
		if d.RuleID == "operation-id" {
			hasOpIDWarning = true
		}
	}
	if !hasOpIDWarning {
		t.Error("expected warning about missing operationId")
	}
}

func TestRunInvalidSpec(t *testing.T) {
	data, err := os.ReadFile("../../testdata/invalid.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	rs, _ := GetRuleset("recommended")
	_, runErr := Run(data, "invalid.yaml", rs)
	if runErr == nil {
		t.Error("expected error for invalid spec")
	}
}

func TestRunEmptyPaths(t *testing.T) {
	data := []byte(`openapi: "3.0.3"
info:
  title: Empty
  version: "1.0"
paths: {}
`)
	rs, _ := GetRuleset("recommended")
	result, err := Run(data, "empty.yaml", rs)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	hasPathsWarning := false
	for _, d := range result.Items {
		if d.RuleID == "paths-defined" {
			hasPathsWarning = true
		}
	}
	if !hasPathsWarning {
		t.Error("expected warning about no paths defined")
	}
}

func TestGetRuleset(t *testing.T) {
	tests := []struct {
		name  string
		found bool
	}{
		{"minimal", true},
		{"recommended", true},
		{"strict", true},
		{"nonexistent", false},
	}
	for _, tt := range tests {
		_, ok := GetRuleset(tt.name)
		if ok != tt.found {
			t.Errorf("GetRuleset(%q) found=%v, want %v", tt.name, ok, tt.found)
		}
	}
}

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		d    Diagnostic
		want string
	}{
		{
			Diagnostic{Severity: "error", RuleID: "info-required", Message: "bad", Path: "spec.yaml"},
			"[error] spec.yaml: bad (info-required)",
		},
		{
			Diagnostic{Severity: "warning", RuleID: "paths-defined", Message: "meh"},
			"[warning] meh (paths-defined)",
		},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

func TestResultCounts(t *testing.T) {
	r := &Result{}
	r.add(Diagnostic{Severity: SeverityError, RuleID: "a", Message: "err"})
	r.add(Diagnostic{Severity: SeverityWarning, RuleID: "b", Message: "warn"})
	r.add(Diagnostic{Severity: SeverityInfo, RuleID: "c", Message: "info"})
	r.add(Diagnostic{Severity: SeverityWarning, RuleID: "d", Message: "warn2"})

	if r.Errors != 1 {
		t.Errorf("Errors = %d, want 1", r.Errors)
	}
	if r.Warnings != 2 {
		t.Errorf("Warnings = %d, want 2", r.Warnings)
	}
	if r.Infos != 1 {
		t.Errorf("Infos = %d, want 1", r.Infos)
	}
	if len(r.Items) != 4 {
		t.Errorf("Items = %d, want 4", len(r.Items))
	}
}

func TestRuleSeverityOff(t *testing.T) {
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
	// minimal turns off operation-id
	rs, _ := GetRuleset("minimal")
	result, err := Run(data, "test.yaml", rs)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	for _, d := range result.Items {
		if d.RuleID == "operation-id" {
			t.Error("minimal ruleset should not report operation-id")
		}
	}
}
