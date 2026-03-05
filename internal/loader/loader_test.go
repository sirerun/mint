package loader

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLoadYAMLFile(t *testing.T) {
	result, err := Load("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if result.Model == nil {
		t.Fatal("expected non-nil model")
	}
	if result.Model.Info.Title != "Petstore" {
		t.Errorf("title = %q, want %q", result.Model.Info.Title, "Petstore")
	}
	if result.Model.Paths.PathItems.Len() < 1 {
		t.Error("expected at least one path")
	}
}

func TestLoadJSONFile(t *testing.T) {
	result, err := Load("../../testdata/petstore.json")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if result.Model == nil {
		t.Fatal("expected non-nil model")
	}
	if result.Model.Info.Title != "Petstore" {
		t.Errorf("title = %q, want %q", result.Model.Info.Title, "Petstore")
	}
}

func TestLoadInvalidFile(t *testing.T) {
	_, err := Load("../../testdata/invalid.yaml")
	if err == nil {
		t.Fatal("expected error for invalid spec")
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("../../testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadURL(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	result, err := Load(srv.URL + "/petstore.yaml")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if result.Model == nil {
		t.Fatal("expected non-nil model")
	}
	if result.Model.Info.Title != "Petstore" {
		t.Errorf("title = %q, want %q", result.Model.Info.Title, "Petstore")
	}
}

func TestLoadReader(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading test file: %v", err)
	}

	result, err := LoadReader(strings.NewReader(string(data)), "<test>")
	if err != nil {
		t.Fatalf("LoadReader() error: %v", err)
	}
	if result.Model == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  Error
		want string
	}{
		{
			name: "with line",
			err:  Error{Message: "bad field", Line: 10, Column: 5, Path: "spec.yaml"},
			want: "spec.yaml:10:5: bad field",
		},
		{
			name: "with path only",
			err:  Error{Message: "parse error", Path: "spec.yaml"},
			want: "spec.yaml: parse error",
		},
		{
			name: "message only",
			err:  Error{Message: "something went wrong"},
			want: "something went wrong",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
