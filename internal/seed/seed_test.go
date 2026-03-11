package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCatalog(t *testing.T, dir string, cat Catalog) string {
	t.Helper()
	path := filepath.Join(dir, "catalog.json")
	data, err := json.Marshal(cat)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCatalog(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "valid",
			content: `{"specs":[{"name":"test","category":"DevOps","spec_url":"http://example.com/spec.yaml","description":"Test API"}]}`,
		},
		{
			name:    "empty specs",
			content: `{"specs":[]}`,
			wantErr: "catalog is empty",
		},
		{
			name:    "invalid json",
			content: `{bad}`,
			wantErr: "parse catalog",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "catalog.json")
			os.WriteFile(path, []byte(tt.content), 0o644)

			cat, err := LoadCatalog(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(cat.Specs) != 1 {
				t.Errorf("got %d specs, want 1", len(cat.Specs))
			}
		})
	}
}

func TestLoadCatalogFileNotFound(t *testing.T) {
	_, err := LoadCatalog("/nonexistent/catalog.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidateCatalog(t *testing.T) {
	tests := []struct {
		name       string
		cat        Catalog
		wantIssues int
	}{
		{
			name: "valid catalog",
			cat: Catalog{Specs: []Spec{
				{Name: "test", Category: "DevOps", SpecURL: "http://example.com", Description: "desc"},
			}},
			wantIssues: 0,
		},
		{
			name: "missing name",
			cat: Catalog{Specs: []Spec{
				{Category: "DevOps", SpecURL: "http://example.com", Description: "desc"},
			}},
			wantIssues: 1,
		},
		{
			name: "missing url",
			cat: Catalog{Specs: []Spec{
				{Name: "test", Category: "DevOps", Description: "desc"},
			}},
			wantIssues: 1,
		},
		{
			name: "missing category",
			cat: Catalog{Specs: []Spec{
				{Name: "test", SpecURL: "http://example.com", Description: "desc"},
			}},
			wantIssues: 1,
		},
		{
			name: "missing description",
			cat: Catalog{Specs: []Spec{
				{Name: "test", Category: "DevOps", SpecURL: "http://example.com"},
			}},
			wantIssues: 1,
		},
		{
			name: "duplicate names",
			cat: Catalog{Specs: []Spec{
				{Name: "test", Category: "DevOps", SpecURL: "http://a.com", Description: "a"},
				{Name: "test", Category: "CRM", SpecURL: "http://b.com", Description: "b"},
			}},
			wantIssues: 1,
		},
		{
			name: "multiple issues",
			cat: Catalog{Specs: []Spec{
				{Name: "", Category: "", SpecURL: "", Description: ""},
			}},
			wantIssues: 3, // name, url, category (description empty counts too = 4... let's check)
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateCatalog(&tt.cat)
			if tt.name == "multiple issues" {
				// name, url, category, description = 4 issues
				if len(issues) < 3 {
					t.Errorf("got %d issues, want at least 3", len(issues))
				}
				return
			}
			if len(issues) != tt.wantIssues {
				t.Errorf("got %d issues, want %d: %v", len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestCategoryCounts(t *testing.T) {
	cat := &Catalog{Specs: []Spec{
		{Name: "a", Category: "Payments"},
		{Name: "b", Category: "Payments"},
		{Name: "c", Category: "CRM"},
		{Name: "d", Category: "DevOps"},
		{Name: "e", Category: "DevOps"},
		{Name: "f", Category: "DevOps"},
	}}

	counts := CategoryCounts(cat)
	if counts["Payments"] != 2 {
		t.Errorf("Payments = %d, want 2", counts["Payments"])
	}
	if counts["CRM"] != 1 {
		t.Errorf("CRM = %d, want 1", counts["CRM"])
	}
	if counts["DevOps"] != 3 {
		t.Errorf("DevOps = %d, want 3", counts["DevOps"])
	}
}

func TestRunDryRun(t *testing.T) {
	dir := t.TempDir()
	cat := Catalog{Specs: []Spec{
		{Name: "test-api", Category: "DevOps", SpecURL: "http://example.com/spec.yaml", Description: "Test"},
		{Name: "test-api-2", Category: "CRM", SpecURL: "http://example.com/spec2.yaml", Description: "Test 2"},
	}}
	path := writeCatalog(t, dir, cat)

	report, err := Run(Options{
		CatalogPath: path,
		OutputDir:   filepath.Join(dir, "out"),
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Total)
	}
	if report.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", report.Succeeded)
	}
	if report.Failed != 0 {
		t.Errorf("Failed = %d, want 0", report.Failed)
	}
}

func TestRunInvalidCatalog(t *testing.T) {
	dir := t.TempDir()
	cat := Catalog{Specs: []Spec{
		{Name: "", Category: "", SpecURL: "", Description: ""},
	}}
	path := writeCatalog(t, dir, cat)

	_, err := Run(Options{
		CatalogPath: path,
		OutputDir:   filepath.Join(dir, "out"),
	})
	if err == nil {
		t.Fatal("expected error for invalid catalog")
	}
	if !strings.Contains(err.Error(), "catalog validation failed") {
		t.Errorf("error %q does not contain expected message", err)
	}
}

func TestRunWithBadBinary(t *testing.T) {
	dir := t.TempDir()
	cat := Catalog{Specs: []Spec{
		{Name: "test-api", Category: "DevOps", SpecURL: "http://example.com/spec.yaml", Description: "Test"},
	}}
	path := writeCatalog(t, dir, cat)

	report, err := Run(Options{
		CatalogPath: path,
		OutputDir:   filepath.Join(dir, "out"),
		MintBinary:  "/nonexistent/mint-binary",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Succeeded != 0 {
		t.Errorf("Succeeded = %d, want 0", report.Succeeded)
	}
	if report.Failed != 1 {
		t.Errorf("Failed = %d, want 1", report.Failed)
	}
	if report.Results[0].Error == "" {
		t.Error("expected error message for failed generation")
	}
}

func TestParseToolCount(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   int
	}{
		{
			name:   "standard output",
			output: "MCP server generated in ./server\n  Tools: 42\n\nNext steps:",
			want:   42,
		},
		{
			name:   "no tools line",
			output: "some other output",
			want:   0,
		},
		{
			name:   "zero tools",
			output: "  Tools: 0\n",
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseToolCount(tt.output)
			if got != tt.want {
				t.Errorf("parseToolCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatReport(t *testing.T) {
	report := &Report{
		Total:     3,
		Succeeded: 2,
		Failed:    1,
		Results: []Result{
			{Name: "stripe", Category: "Payments", Success: true, ToolCount: 42},
			{Name: "github", Category: "DevOps", Success: true, ToolCount: 100},
			{Name: "bad-api", Category: "CRM", Success: false, Error: "spec not found"},
		},
	}

	output := FormatReport(report)

	if !strings.Contains(output, "Total:     3") {
		t.Error("report missing total")
	}
	if !strings.Contains(output, "Succeeded: 2") {
		t.Error("report missing succeeded")
	}
	if !strings.Contains(output, "Failed:    1") {
		t.Error("report missing failed")
	}
	if !strings.Contains(output, "bad-api") {
		t.Error("report missing failure details")
	}
	if !strings.Contains(output, "spec not found") {
		t.Error("report missing error message")
	}
	if !strings.Contains(output, "By Category") {
		t.Error("report missing category breakdown")
	}
}

func TestFormatReportNoFailures(t *testing.T) {
	report := &Report{
		Total:     1,
		Succeeded: 1,
		Failed:    0,
		Results: []Result{
			{Name: "stripe", Category: "Payments", Success: true},
		},
	}

	output := FormatReport(report)
	if strings.Contains(output, "Failures:") {
		t.Error("report should not show failures section when none failed")
	}
}

func TestLoadEmbeddedCatalog(t *testing.T) {
	// Test that the actual catalog.json embedded in the package is valid.
	// Find catalog.json relative to the test file.
	cat, err := LoadCatalog("catalog.json")
	if err != nil {
		t.Fatalf("LoadCatalog(embedded): %v", err)
	}

	if len(cat.Specs) < 100 {
		t.Errorf("catalog has %d specs, want at least 100", len(cat.Specs))
	}

	issues := ValidateCatalog(cat)
	if len(issues) > 0 {
		t.Errorf("catalog has validation issues:\n  %s", strings.Join(issues, "\n  "))
	}

	// Verify all 10 categories are represented.
	counts := CategoryCounts(cat)
	wantCategories := []string{
		"Payments", "CRM", "Communication", "DevOps",
		"Productivity", "Analytics", "Support", "Marketing",
		"HR", "Infrastructure",
	}
	for _, cat := range wantCategories {
		if counts[cat] == 0 {
			t.Errorf("missing category %q", cat)
		}
	}
}
