package managed

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCreateSourceTarball(t *testing.T) {
	dir := t.TempDir()

	// Create a test directory structure.
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "helper.go"), []byte("package sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := CreateSourceTarball(dir, &buf); err != nil {
		t.Fatalf("CreateSourceTarball: %v", err)
	}

	// Verify the tarball contents.
	gr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	var names []string
	contents := make(map[string]string)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next: %v", err)
		}
		names = append(names, hdr.Name)

		if !hdr.FileInfo().IsDir() {
			data, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("reading %s: %v", hdr.Name, err)
			}
			contents[hdr.Name] = string(data)
		}
	}

	sort.Strings(names)
	want := []string{"main.go", "sub/", "sub/helper.go"}
	if len(names) != len(want) {
		t.Fatalf("tar entries = %v, want %v", names, want)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("entry[%d] = %q, want %q", i, name, want[i])
		}
	}

	if contents["main.go"] != "package main\n" {
		t.Errorf("main.go content = %q", contents["main.go"])
	}
	if contents["sub/helper.go"] != "package sub\n" {
		t.Errorf("sub/helper.go content = %q", contents["sub/helper.go"])
	}
}

func TestCreateSourceTarballEmptyDir(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	if err := CreateSourceTarball(dir, &buf); err != nil {
		t.Fatalf("CreateSourceTarball: %v", err)
	}

	// Should produce a valid but empty tarball.
	gr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	_, err = tr.Next()
	if err != io.EOF {
		t.Fatalf("expected EOF for empty dir, got: %v", err)
	}
}

func TestCreateSourceTarballInvalidDir(t *testing.T) {
	err := CreateSourceTarball("/nonexistent/path/that/does/not/exist", io.Discard)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestUploadSource(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/sources" {
			t.Errorf("expected /sources, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		// Verify multipart form.
		err := r.ParseMultipartForm(1 << 20)
		if err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		file, _, err := r.FormFile("source")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()

		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("reading file: %v", err)
		}
		if string(data) != "fake-tarball-content" {
			t.Errorf("unexpected file content: %q", string(data))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("src-abc123"))
	}))
	defer srv.Close()

	hc := &httpClient{
		baseURL:    srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	content := []byte("fake-tarball-content")
	var stderr bytes.Buffer
	sourceID, err := uploadSource(context.Background(), hc, bytes.NewReader(content), int64(len(content)), &stderr)
	if err != nil {
		t.Fatalf("uploadSource: %v", err)
	}
	if sourceID != "src-abc123" {
		t.Errorf("sourceID = %q, want %q", sourceID, "src-abc123")
	}
	if stderr.Len() == 0 {
		t.Error("expected progress output on stderr")
	}
}

func TestUploadSourceHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	hc := &httpClient{
		baseURL:    srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	var stderr bytes.Buffer
	_, err := uploadSource(context.Background(), hc, bytes.NewReader([]byte("data")), 4, &stderr)
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
}

func TestUploadSourceZeroSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("src-empty"))
	}))
	defer srv.Close()

	hc := &httpClient{
		baseURL:    srv.URL,
		token:      "test-token",
		httpClient: &http.Client{},
	}

	var stderr bytes.Buffer
	sourceID, err := uploadSource(context.Background(), hc, bytes.NewReader(nil), 0, &stderr)
	if err != nil {
		t.Fatalf("uploadSource: %v", err)
	}
	if sourceID != "src-empty" {
		t.Errorf("sourceID = %q, want %q", sourceID, "src-empty")
	}
}
