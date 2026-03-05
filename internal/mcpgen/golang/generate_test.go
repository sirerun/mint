package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sirerun/mint/internal/loader"
	"github.com/sirerun/mint/internal/mcpgen"
)

func TestGenerate(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Check all expected files exist
	expectedFiles := []string{
		"main.go", "server.go", "tools.go", "client.go", "go.mod", "README.md",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", f)
		}
	}
}

func TestGenerateCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compilation test in short mode")
	}

	// Check go is available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not found in PATH")
	}

	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	// Run go build
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}

func TestExportName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"list_pets", "ListPets"},
		{"create_pet", "CreatePet"},
		{"show_pet_by_id", "ShowPetById"},
		{"simple", "Simple"},
		{"a_b_c", "ABC"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := exportName(tt.input); got != tt.want {
				t.Errorf("exportName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHasBodyParams(t *testing.T) {
	tests := []struct {
		name   string
		params []mcpgen.MCPToolParam
		want   bool
	}{
		{
			name:   "no params",
			params: nil,
			want:   false,
		},
		{
			name:   "query params only",
			params: []mcpgen.MCPToolParam{{In: "query"}},
			want:   false,
		},
		{
			name:   "has body param",
			params: []mcpgen.MCPToolParam{{In: "body"}},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasBodyParams(tt.params); got != tt.want {
				t.Errorf("hasBodyParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
