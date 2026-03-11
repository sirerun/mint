package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/sirerun/mint/registry/db"
	"github.com/sirerun/mint/registry/handler"
	"github.com/sirerun/mint/registry/model"
)

func setupServer(t *testing.T) (*httptest.Server, *db.DB) {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	h := &handler.Handler{
		DB:          store,
		ArtifactDir: t.TempDir(),
	}
	mux := buildMux(h, store)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, store
}

func TestHealthCheck(t *testing.T) {
	ts, _ := setupServer(t)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	ts, _ := setupServer(t)
	client := ts.Client()

	// 1. Register publisher.
	body := bytes.NewBufferString(`{"github_handle":"alice"}`)
	resp, err := client.Post(ts.URL+"/api/v1/publishers/register", "application/json", body)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("register status = %d; body: %s", resp.StatusCode, b)
	}

	var regResp map[string]string
	json.NewDecoder(resp.Body).Decode(&regResp)
	apiKey := regResp["api_key"]
	if apiKey == "" {
		t.Fatal("expected api_key in registration response")
	}

	// 2. Publish a server.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	meta := model.PublishRequest{
		Name:        "test-server",
		Description: "Integration test server",
		Version:     "1.0.0",
		Category:    "testing",
	}
	metaJSON, _ := json.Marshal(meta)
	writer.WriteField("metadata", string(metaJSON))
	part, _ := writer.CreateFormFile("artifact", "server.tar.gz")
	part.Write([]byte("fake tarball"))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("publish status = %d; body: %s", resp.StatusCode, b)
	}

	var pubResp map[string]string
	json.NewDecoder(resp.Body).Decode(&pubResp)
	serverID := pubResp["server_id"]

	// 3. List servers.
	resp, err = client.Get(ts.URL + "/api/v1/servers?q=test")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("list status = %d", resp.StatusCode)
	}
	var listResp model.ServerListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)
	if listResp.Total != 1 {
		t.Errorf("total = %d, want 1", listResp.Total)
	}

	// 4. Get server detail.
	resp, err = client.Get(ts.URL + "/api/v1/servers/" + serverID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get status = %d", resp.StatusCode)
	}

	// 5. Download.
	resp, err = client.Get(ts.URL + "/api/v1/servers/" + serverID + "/download")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("download status = %d; body: %s", resp.StatusCode, b)
	}

	// 6. Star (authenticated).
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/v1/servers/"+serverID+"/star", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("star: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("star status = %d; body: %s", resp.StatusCode, b)
	}

	// 7. Get stars.
	resp, err = client.Get(ts.URL + "/api/v1/servers/" + serverID + "/stars")
	if err != nil {
		t.Fatalf("get stars: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("get stars status = %d", resp.StatusCode)
	}
	var starsResp map[string]any
	json.NewDecoder(resp.Body).Decode(&starsResp)
	if starsResp["stars"] != float64(1) {
		t.Errorf("stars = %v, want 1", starsResp["stars"])
	}
}

func TestIntegration_PublishNoAuth(t *testing.T) {
	ts, _ := setupServer(t)
	client := ts.Client()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("metadata", `{"name":"test","description":"d","version":"1.0.0"}`)
	part, _ := writer.CreateFormFile("artifact", "s.tar.gz")
	part.Write([]byte("content"))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestRouteServerSubpath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		suffix string
		match  bool
	}{
		{"download", "/api/v1/servers/abc/download", "download", true},
		{"star", "/api/v1/servers/abc/star", "star", true},
		{"stars", "/api/v1/servers/abc/stars", "stars", true},
		{"unknown suffix mismatch", "/api/v1/servers/abc/unknown", "download", false},
		{"no subpath", "/api/v1/servers/abc", "download", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, _ := matchSubpath(tt.path, tt.suffix)
			if matched != tt.match {
				t.Errorf("matchSubpath(%q, %q) = %v, want %v", tt.path, tt.suffix, matched, tt.match)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"abc/def", 2},
		{"abc", 1},
		{"abc/def/ghi", 3},
		{"", 0},
	}

	for _, tt := range tests {
		parts := splitPath(tt.input)
		if len(parts) != tt.want {
			t.Errorf("splitPath(%q) = %d parts, want %d", tt.input, len(parts), tt.want)
		}
	}
}

func TestChain(t *testing.T) {
	var order []string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1")
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2")
			next.ServeHTTP(w, r)
		})
	}

	wrapped := chain(h, mw1, mw2)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	if len(order) != 3 || order[0] != "mw1" || order[1] != "mw2" || order[2] != "handler" {
		t.Errorf("order = %v, want [mw1 mw2 handler]", order)
	}
}
