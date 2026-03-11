package publish

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadManifest(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "valid manifest",
			content: `{"name":"test-server","version":"1.0.0","description":"A test server"}`,
		},
		{
			name:    "missing name",
			content: `{"version":"1.0.0","description":"A test server"}`,
			wantErr: "name is required",
		},
		{
			name:    "missing version",
			content: `{"name":"test-server","description":"A test server"}`,
			wantErr: "version is required",
		},
		{
			name:    "missing description",
			content: `{"name":"test-server","version":"1.0.0"}`,
			wantErr: "description is required",
		},
		{
			name:    "invalid json",
			content: `{invalid}`,
			wantErr: "parse mint.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.content != "" {
				if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(tt.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			m, err := ReadManifest(dir)
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
			if m.Name != "test-server" {
				t.Errorf("Name = %q, want %q", m.Name, "test-server")
			}
		})
	}
}

func TestReadManifestNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadManifest(dir)
	if err == nil {
		t.Fatal("expected error for missing mint.json")
	}
	if !strings.Contains(err.Error(), "mint.json not found") {
		t.Fatalf("error %q does not mention mint.json not found", err)
	}
}

func TestPackageTarball(t *testing.T) {
	dir := t.TempDir()

	// Create some files.
	if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create excluded directories.
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref"), 0o644); err != nil {
		t.Fatal(err)
	}

	nodeDir := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "pkg.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	buf, err := PackageTarball(dir)
	if err != nil {
		t.Fatalf("PackageTarball: %v", err)
	}

	// Extract and verify contents.
	gr, err := gzip.NewReader(buf)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	var files []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		files = append(files, hdr.Name)
	}

	// Should contain mint.json and main.go but not .git/ or node_modules/.
	hasMain := false
	hasGit := false
	hasNodeModules := false
	for _, f := range files {
		if strings.Contains(f, "main.go") {
			hasMain = true
		}
		if strings.Contains(f, ".git") {
			hasGit = true
		}
		if strings.Contains(f, "node_modules") {
			hasNodeModules = true
		}
	}
	if !hasMain {
		t.Error("tarball missing main.go")
	}
	if hasGit {
		t.Error("tarball should exclude .git")
	}
	if hasNodeModules {
		t.Error("tarball should exclude node_modules")
	}
}

func TestUploadDryRun(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"test-server","version":"1.2.3","description":"Test"}`
	if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := Upload(Options{
		Dir:    dir,
		DryRun: true,
		Token:  "test-token",
	})
	if err != nil {
		t.Fatalf("Upload dry run: %v", err)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", resp.Version, "1.2.3")
	}
}

func TestUploadSuccess(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"test-server","version":"1.0.0","description":"A test server"}`
	if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("content-type = %s, want multipart", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("auth header = %q, want %q", r.Header.Get("Authorization"), "Bearer test-token")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"server_id":  "srv-123",
			"version_id": "ver-456",
			"version":    "1.0.0",
			"checksum":   "abc123",
		})
	}))
	defer srv.Close()

	resp, err := Upload(Options{
		Dir:         dir,
		RegistryURL: srv.URL,
		Token:       "test-token",
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if resp.ServerID != "srv-123" {
		t.Errorf("ServerID = %q, want %q", resp.ServerID, "srv-123")
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{name: "valid", m: Manifest{Name: "x", Version: "1.0.0", Description: "d"}},
		{name: "no name", m: Manifest{Version: "1.0.0", Description: "d"}, wantErr: true},
		{name: "no version", m: Manifest{Name: "x", Description: "d"}, wantErr: true},
		{name: "no desc", m: Manifest{Name: "x", Version: "1.0.0"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.m.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadMissingManifest(t *testing.T) {
	dir := t.TempDir()
	_, err := Upload(Options{Dir: dir, Token: "test"})
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
}

func TestPackageTarballSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory to verify directory traversal.
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}

	buf, err := PackageTarball(dir)
	if err != nil {
		t.Fatalf("PackageTarball: %v", err)
	}

	// Verify the tarball has contents.
	gr, err := gzip.NewReader(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	count := 0
	for {
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count == 0 {
		t.Error("tarball is empty")
	}
}

func TestPackageTarballExcludesEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	buf, err := PackageTarball(dir)
	if err != nil {
		t.Fatalf("PackageTarball: %v", err)
	}

	gr, err := gzip.NewReader(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(hdr.Name, ".env") {
			t.Error("tarball should exclude .env")
		}
		if strings.Contains(hdr.Name, ".DS_Store") {
			t.Error("tarball should exclude .DS_Store")
		}
	}
}

func TestUploadNonJSONErrorResponse(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"test-server","version":"1.0.0","description":"A test server"}`
	if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	_, err := Upload(Options{
		Dir:         dir,
		RegistryURL: srv.URL,
		Token:       "test-token",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "internal server error") {
		t.Errorf("error %q should contain raw response", err)
	}
}

func TestUploadServerError(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"test-server","version":"1.0.0","description":"A test server"}`
	if err := os.WriteFile(filepath.Join(dir, "mint.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "version already exists"})
	}))
	defer srv.Close()

	_, err := Upload(Options{
		Dir:         dir,
		RegistryURL: srv.URL,
		Token:       "test-token",
	})
	if err == nil {
		t.Fatal("expected error for conflict response")
	}
	if !strings.Contains(err.Error(), "version already exists") {
		t.Errorf("error %q does not contain expected message", err)
	}
}
