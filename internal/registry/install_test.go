package registry

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_Success(t *testing.T) {
	specContent := "openapi: '3.0.0'\ninfo:\n  title: Test\n  version: '1.0'"
	specSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte(specContent))
	}))
	defer specSrv.Close()

	idx := &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{
				Name:       "test-api",
				SpecURL:    specSrv.URL,
				AuthType:   "bearer",
				AuthEnvVar: "TEST_API_TOKEN",
			},
		},
	}

	outputDir := filepath.Join(t.TempDir(), "test-api")
	var stderr bytes.Buffer

	err := Install(context.Background(), idx, InstallOptions{
		Name:      "test-api",
		OutputDir: outputDir,
	}, &stderr)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Verify spec file was written.
	data, err := os.ReadFile(filepath.Join(outputDir, "openapi.yaml"))
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	if string(data) != specContent {
		t.Errorf("spec content = %q, want %q", string(data), specContent)
	}

	// Verify stderr output.
	output := stderr.String()
	if !strings.Contains(output, "Fetching spec from") {
		t.Error("missing 'Fetching spec from' message")
	}
	if !strings.Contains(output, "mint mcp generate") {
		t.Error("missing generate instruction")
	}
}

func TestInstall_DefaultOutputDir(t *testing.T) {
	specSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("spec"))
	}))
	defer specSrv.Close()

	// We need to change to a temp directory so the default output dir is created there.
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	idx := &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "my-api", SpecURL: specSrv.URL},
		},
	}

	var stderr bytes.Buffer
	err := Install(context.Background(), idx, InstallOptions{Name: "my-api"}, &stderr)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if _, err := os.Stat(filepath.Join("my-api", "openapi.yaml")); err != nil {
		t.Errorf("spec file not created at default output dir: %v", err)
	}
}

func TestInstall_EntryNotFound(t *testing.T) {
	idx := &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "github", SpecURL: "http://example.com/spec.yaml"},
		},
	}

	var stderr bytes.Buffer
	err := Install(context.Background(), idx, InstallOptions{Name: "nonexistent"}, &stderr)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}

func TestInstall_DownloadFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	idx := &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "fail-api", SpecURL: srv.URL},
		},
	}

	var stderr bytes.Buffer
	err := Install(context.Background(), idx, InstallOptions{
		Name:      "fail-api",
		OutputDir: t.TempDir(),
	}, &stderr)
	if err == nil {
		t.Fatal("expected error for download failure")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("error = %q, want HTTP 500", err.Error())
	}
}

func TestInstall_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never responds.
		select {}
	}))
	defer srv.Close()

	idx := &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "cancel-api", SpecURL: srv.URL},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stderr bytes.Buffer
	err := Install(ctx, idx, InstallOptions{
		Name:      "cancel-api",
		OutputDir: t.TempDir(),
	}, &stderr)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestFormatPostInstall_WithAuth(t *testing.T) {
	entry := RegistryEntry{
		Name:       "github",
		AuthType:   "bearer",
		AuthEnvVar: "GITHUB_TOKEN",
	}
	out := FormatPostInstall(entry, "output")
	if !strings.Contains(out, "openapi.yaml") {
		t.Error("missing spec file path")
	}
	if !strings.Contains(out, "mint mcp generate") {
		t.Error("missing generate command")
	}
	if !strings.Contains(out, "export GITHUB_TOKEN") {
		t.Error("missing auth env var instruction")
	}
	if !strings.Contains(out, "go build") {
		t.Error("missing build instruction")
	}
}

func TestFormatPostInstall_NoAuth(t *testing.T) {
	entry := RegistryEntry{
		Name: "public-api",
	}
	out := FormatPostInstall(entry, "output")
	if strings.Contains(out, "export") {
		t.Error("should not contain auth instruction when no AuthEnvVar")
	}
	if !strings.Contains(out, "mint mcp generate") {
		t.Error("missing generate command")
	}
}
