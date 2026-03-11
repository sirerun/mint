package install

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNameVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{input: "stripe-mcp", wantName: "stripe-mcp", wantVersion: ""},
		{input: "stripe-mcp@1.2.0", wantName: "stripe-mcp", wantVersion: "1.2.0"},
		{input: "stripe-mcp@latest", wantName: "stripe-mcp", wantVersion: "latest"},
		{input: "@scoped/pkg@1.0.0", wantName: "@scoped/pkg", wantVersion: "1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version := ParseNameVersion(tt.input)
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func createTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: "test-server/" + name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestInstallSuccess(t *testing.T) {
	tarData := createTestTarGz(t, map[string]string{
		"mint.json": `{"name":"test-server"}`,
		"main.go":   "package main",
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/servers" && r.URL.Query().Get("q") == "test-server":
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]string{
					{"id": "srv-123", "name": "test-server"},
				},
				"total": 1,
			})
		case r.URL.Path == "/servers/srv-123/download":
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	installDir := t.TempDir()

	dest, err := Install(Options{
		Name:        "test-server",
		RegistryURL: srv.URL,
		InstallDir:  installDir,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	expectedDir := filepath.Join(installDir, "test-server")
	if dest != expectedDir {
		t.Errorf("dest = %q, want %q", dest, expectedDir)
	}

	// Verify extracted files.
	data, err := os.ReadFile(filepath.Join(expectedDir, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if string(data) != "package main" {
		t.Errorf("main.go content = %q, want %q", string(data), "package main")
	}
}

func TestInstallWithVersion(t *testing.T) {
	tarData := createTestTarGz(t, map[string]string{"mint.json": `{}`})

	var gotVersionQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/servers":
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]string{
					{"id": "srv-123", "name": "stripe-mcp"},
				},
				"total": 1,
			})
		case r.URL.Path == "/servers/srv-123/download":
			gotVersionQuery = r.URL.Query().Get("version")
			w.Header().Set("Content-Type", "application/gzip")
			w.Write(tarData)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, err := Install(Options{
		Name:        "stripe-mcp@1.2.0",
		RegistryURL: srv.URL,
		InstallDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if gotVersionQuery != "1.2.0" {
		t.Errorf("version query = %q, want %q", gotVersionQuery, "1.2.0")
	}
}

func TestInstallServerNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"servers": []map[string]string{},
			"total":   0,
		})
	}))
	defer srv.Close()

	_, err := Install(Options{
		Name:        "nonexistent",
		RegistryURL: srv.URL,
		InstallDir:  t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestInstallDownloadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/servers":
			json.NewEncoder(w).Encode(map[string]any{
				"servers": []map[string]string{
					{"id": "srv-123", "name": "bad-server"},
				},
				"total": 1,
			})
		case r.URL.Path == "/servers/srv-123/download":
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "version not found"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, err := Install(Options{
		Name:        "bad-server",
		RegistryURL: srv.URL,
		InstallDir:  t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for download failure")
	}
}

func TestInstallSearchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := Install(Options{
		Name:        "any-server",
		RegistryURL: srv.URL,
		InstallDir:  t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error for search failure")
	}
}

func TestExtractTarGzWithDirectory(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// Add a directory entry.
	tw.WriteHeader(&tar.Header{
		Name:     "test-server/subdir/",
		Typeflag: tar.TypeDir,
		Mode:     0o755,
	})
	// Add a file inside the directory.
	content := "hello"
	tw.WriteHeader(&tar.Header{
		Name: "test-server/subdir/file.txt",
		Mode: 0o644,
		Size: int64(len(content)),
	})
	tw.Write([]byte(content))
	tw.Close()
	gw.Close()

	destDir := t.TempDir()
	if err := extractTarGz(&buf, destDir); err != nil {
		t.Fatalf("extractTarGz: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(destDir, "subdir", "file.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", string(data), content)
	}
}

func TestExtractTarGzPathTraversal(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: "../../etc/passwd",
		Mode: 0o644,
		Size: 4,
	}
	tw.WriteHeader(hdr)
	tw.Write([]byte("evil"))
	tw.Close()
	gw.Close()

	destDir := t.TempDir()
	err := extractTarGz(&buf, destDir)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}
