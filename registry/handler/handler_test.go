package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/sirerun/mint/registry/db"
	"github.com/sirerun/mint/registry/middleware"
	"github.com/sirerun/mint/registry/model"
)

func setupTestHandler(t *testing.T) (*Handler, *db.DB) {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	artifactDir := t.TempDir()
	h := &Handler{
		DB:          store,
		ArtifactDir: artifactDir,
	}
	return h, store
}

func createTestPublisher(t *testing.T, store *db.DB, handle string) (*model.Publisher, string) {
	t.Helper()
	apiKey := "test-key-" + handle
	pub := &model.Publisher{
		ID:           "pub-" + handle,
		GithubHandle: handle,
		APIKeyHash:   db.HashAPIKey(apiKey),
	}
	if err := store.CreatePublisher(pub); err != nil {
		t.Fatalf("create publisher: %v", err)
	}
	return pub, apiKey
}

func withAuth(r *http.Request, pub *model.Publisher) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.PublisherContextKey(), pub)
	return r.WithContext(ctx)
}

func buildPublishRequest(t *testing.T, metadata model.PublishRequest, artifactContent []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	metaJSON, _ := json.Marshal(metadata)
	writer.WriteField("metadata", string(metaJSON))

	part, err := writer.CreateFormFile("artifact", "server.tar.gz")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	part.Write(artifactContent)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/publish", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestPublish_Success(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	meta := model.PublishRequest{
		Name:        "my-server",
		Description: "A great MCP server",
		Version:     "1.0.0",
		Category:    "testing",
	}
	req := buildPublishRequest(t, meta, []byte("fake tarball content"))
	req = withAuth(req, pub)

	w := httptest.NewRecorder()
	h.Publish(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["server_id"] == "" {
		t.Error("expected server_id in response")
	}
	if resp["version"] != "1.0.0" {
		t.Errorf("version = %q, want %q", resp["version"], "1.0.0")
	}
	if resp["checksum"] == "" {
		t.Error("expected checksum in response")
	}

	// Verify artifact was stored.
	artifactPath := filepath.Join(h.ArtifactDir, resp["server_id"], "1.0.0.tar.gz")
	if _, err := os.Stat(artifactPath); err != nil {
		t.Errorf("artifact file not found: %v", err)
	}
}

func TestPublish_NewVersion(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	// Publish v1.
	meta := model.PublishRequest{
		Name: "my-server", Description: "desc", Version: "1.0.0",
	}
	req := buildPublishRequest(t, meta, []byte("v1"))
	req = withAuth(req, pub)
	w := httptest.NewRecorder()
	h.Publish(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("v1 publish status = %d", w.Code)
	}

	// Publish v2.
	meta.Version = "2.0.0"
	req = buildPublishRequest(t, meta, []byte("v2"))
	req = withAuth(req, pub)
	w = httptest.NewRecorder()
	h.Publish(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("v2 publish status = %d; body: %s", w.Code, w.Body.String())
	}
}

func TestPublish_DuplicateVersion(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	meta := model.PublishRequest{
		Name: "my-server", Description: "desc", Version: "1.0.0",
	}
	req := buildPublishRequest(t, meta, []byte("v1"))
	req = withAuth(req, pub)
	w := httptest.NewRecorder()
	h.Publish(w, req)

	// Publish same version again.
	req = buildPublishRequest(t, meta, []byte("v1 again"))
	req = withAuth(req, pub)
	w = httptest.NewRecorder()
	h.Publish(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestPublish_InvalidName(t *testing.T) {
	tests := []struct {
		name    string
		srvName string
	}{
		{"too short", "ab"},
		{"uppercase", "MyServer"},
		{"starts with number", "1server"},
		{"spaces", "my server"},
		{"ends with hyphen", "my-server-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, store := setupTestHandler(t)
			pub, _ := createTestPublisher(t, store, "alice")

			meta := model.PublishRequest{
				Name: tt.srvName, Description: "desc", Version: "1.0.0",
			}
			req := buildPublishRequest(t, meta, []byte("content"))
			req = withAuth(req, pub)
			w := httptest.NewRecorder()
			h.Publish(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d for name %q", w.Code, http.StatusBadRequest, tt.srvName)
			}
		})
	}
}

func TestPublish_InvalidVersion(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	meta := model.PublishRequest{
		Name: "my-server", Description: "desc", Version: "not-semver",
	}
	req := buildPublishRequest(t, meta, []byte("content"))
	req = withAuth(req, pub)
	w := httptest.NewRecorder()
	h.Publish(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPublish_Unauthorized(t *testing.T) {
	h, _ := setupTestHandler(t)

	meta := model.PublishRequest{
		Name: "my-server", Description: "desc", Version: "1.0.0",
	}
	req := buildPublishRequest(t, meta, []byte("content"))
	// No auth context.
	w := httptest.NewRecorder()
	h.Publish(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestPublish_WrongOwner(t *testing.T) {
	h, store := setupTestHandler(t)
	alice, _ := createTestPublisher(t, store, "alice")
	bob, _ := createTestPublisher(t, store, "bob")

	// Alice publishes.
	meta := model.PublishRequest{
		Name: "my-server", Description: "desc", Version: "1.0.0",
	}
	req := buildPublishRequest(t, meta, []byte("content"))
	req = withAuth(req, alice)
	w := httptest.NewRecorder()
	h.Publish(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("alice publish status = %d", w.Code)
	}

	// Bob tries to publish to same name.
	meta.Version = "2.0.0"
	req = buildPublishRequest(t, meta, []byte("bob's content"))
	req = withAuth(req, bob)
	w = httptest.NewRecorder()
	h.Publish(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestListServers(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	for _, name := range []string{"stripe-mcp", "github-mcp", "slack-mcp"} {
		srv := &model.Server{
			ID: "id-" + name, Name: name, Description: "Server for " + name,
			PublisherID: pub.ID, LatestVersion: "1.0.0", Category: "api",
		}
		store.CreateServer(srv)
	}

	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"all", "", 3},
		{"search stripe", "q=stripe", 1},
		{"search mcp", "q=mcp", 3},
		{"search nonexistent", "q=nonexistent", 0},
		{"filter category", "category=api", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/servers?"+tt.query, nil)
			w := httptest.NewRecorder()
			h.ListServers(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d", w.Code)
			}

			var resp model.ServerListResponse
			json.NewDecoder(w.Body).Decode(&resp)
			if len(resp.Servers) != tt.wantLen {
				t.Errorf("got %d servers, want %d", len(resp.Servers), tt.wantLen)
			}
		})
	}
}

func TestGetServer(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	srv := &model.Server{
		ID: "srv-1", Name: "test-server", Description: "Test",
		PublisherID: pub.ID, LatestVersion: "1.0.0",
	}
	store.CreateServer(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1", nil)
	w := httptest.NewRecorder()
	h.GetServer(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp map[string]json.RawMessage
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["server"]; !ok {
		t.Error("expected 'server' in response")
	}
}

func TestGetServer_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/nonexistent", nil)
	w := httptest.NewRecorder()
	h.GetServer(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDownload(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	srv := &model.Server{
		ID: "srv-1", Name: "test-server", Description: "Test",
		PublisherID: pub.ID, LatestVersion: "1.0.0",
	}
	store.CreateServer(srv)

	// Create artifact file.
	artifactDir := filepath.Join(h.ArtifactDir, "srv-1")
	os.MkdirAll(artifactDir, 0o755)
	content := []byte("tarball content")
	os.WriteFile(filepath.Join(artifactDir, "1.0.0.tar.gz"), content, 0o644)

	ver := &model.Version{
		ID: "v1", ServerID: "srv-1", Version: "1.0.0",
		ArtifactPath: filepath.Join(artifactDir, "1.0.0.tar.gz"),
		Checksum:     "abc123",
	}
	store.CreateVersion(ver)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1/download?version=1.0.0", nil)
	w := httptest.NewRecorder()
	h.Download(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}

	body, _ := io.ReadAll(w.Body)
	if string(body) != string(content) {
		t.Errorf("body = %q, want %q", body, content)
	}

	// Check download counter incremented.
	updated, _ := store.GetServerByID("srv-1")
	if updated.Downloads != 1 {
		t.Errorf("downloads = %d, want 1", updated.Downloads)
	}
}

func TestDownload_Latest(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	srv := &model.Server{
		ID: "srv-1", Name: "test-server", Description: "Test",
		PublisherID: pub.ID, LatestVersion: "2.0.0",
	}
	store.CreateServer(srv)

	artifactDir := filepath.Join(h.ArtifactDir, "srv-1")
	os.MkdirAll(artifactDir, 0o755)
	os.WriteFile(filepath.Join(artifactDir, "1.0.0.tar.gz"), []byte("v1"), 0o644)
	os.WriteFile(filepath.Join(artifactDir, "2.0.0.tar.gz"), []byte("v2"), 0o644)

	store.CreateVersion(&model.Version{
		ID: "v1", ServerID: "srv-1", Version: "1.0.0",
		ArtifactPath: filepath.Join(artifactDir, "1.0.0.tar.gz"),
	})
	store.CreateVersion(&model.Version{
		ID: "v2", ServerID: "srv-1", Version: "2.0.0",
		ArtifactPath: filepath.Join(artifactDir, "2.0.0.tar.gz"),
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1/download", nil)
	w := httptest.NewRecorder()
	h.Download(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestDownload_NotFound(t *testing.T) {
	h, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/nonexistent/download", nil)
	w := httptest.NewRecorder()
	h.Download(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestStarToggle(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	srv := &model.Server{
		ID: "srv-1", Name: "test-server", Description: "Test",
		PublisherID: pub.ID, LatestVersion: "1.0.0",
	}
	store.CreateServer(srv)

	// Star.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/srv-1/star", nil)
	req = withAuth(req, pub)
	w := httptest.NewRecorder()
	h.StarToggle(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["starred"] != true {
		t.Error("expected starred = true")
	}

	// Unstar.
	req = httptest.NewRequest(http.MethodPost, "/api/v1/servers/srv-1/star", nil)
	req = withAuth(req, pub)
	w = httptest.NewRecorder()
	h.StarToggle(w, req)

	json.NewDecoder(w.Body).Decode(&resp)
	if resp["starred"] != false {
		t.Error("expected starred = false")
	}
}

func TestStarToggle_Unauthorized(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	srv := &model.Server{
		ID: "srv-1", Name: "test-server", Description: "Test",
		PublisherID: pub.ID, LatestVersion: "1.0.0",
	}
	store.CreateServer(srv)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/servers/srv-1/star", nil)
	w := httptest.NewRecorder()
	h.StarToggle(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestGetStars(t *testing.T) {
	h, store := setupTestHandler(t)
	pub, _ := createTestPublisher(t, store, "alice")

	srv := &model.Server{
		ID: "srv-1", Name: "test-server", Description: "Test",
		PublisherID: pub.ID, LatestVersion: "1.0.0",
	}
	store.CreateServer(srv)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers/srv-1/stars", nil)
	w := httptest.NewRecorder()
	h.GetStars(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["stars"] != float64(0) {
		t.Errorf("stars = %v, want 0", resp["stars"])
	}
}

func TestRegisterPublisher(t *testing.T) {
	h, _ := setupTestHandler(t)

	body := bytes.NewBufferString(`{"github_handle":"newuser"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishers/register", body)
	w := httptest.NewRecorder()
	h.RegisterPublisher(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["publisher_id"] == "" {
		t.Error("expected publisher_id")
	}
	if resp["api_key"] == "" {
		t.Error("expected api_key")
	}
	if len(resp["api_key"]) < 10 {
		t.Error("api_key seems too short")
	}
}

func TestRegisterPublisher_Duplicate(t *testing.T) {
	h, store := setupTestHandler(t)
	createTestPublisher(t, store, "existing")

	body := bytes.NewBufferString(`{"github_handle":"existing"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishers/register", body)
	w := httptest.NewRecorder()
	h.RegisterPublisher(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestRegisterPublisher_MissingHandle(t *testing.T) {
	h, _ := setupTestHandler(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishers/register", body)
	w := httptest.NewRecorder()
	h.RegisterPublisher(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h, _ := setupTestHandler(t)

	tests := []struct {
		name    string
		method  string
		path    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{"publish GET", http.MethodGet, "/api/v1/publish", h.Publish},
		{"list POST", http.MethodPost, "/api/v1/servers", h.ListServers},
		{"get POST", http.MethodPost, "/api/v1/servers/x", h.GetServer},
		{"download POST", http.MethodPost, "/api/v1/servers/x/download", h.Download},
		{"star GET", http.MethodGet, "/api/v1/servers/x/star", h.StarToggle},
		{"stars POST", http.MethodPost, "/api/v1/servers/x/stars", h.GetStars},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestNameValidation(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"abc", true},
		{"my-server", true},
		{"stripe-mcp-server", true},
		{"a1b2c3", true},
		{"ab", false},  // too short
		{"Ab", false},  // uppercase
		{"1ab", false}, // starts with number
		{"-ab", false}, // starts with hyphen
		{"ab-", false}, // ends with hyphen
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if nameRe.MatchString(tt.name) != tt.valid {
				t.Errorf("nameRe.MatchString(%q) = %v, want %v", tt.name, !tt.valid, tt.valid)
			}
		})
	}
}

func TestSemverValidation(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"1.0.0", true},
		{"v1.0.0", true},
		{"0.1.0", true},
		{"1.2.3-beta.1", true},
		{"1.0.0+build.123", true},
		{"not-semver", false},
		{"1.0", false},
		{"v1", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			if semverRe.MatchString(tt.version) != tt.valid {
				t.Errorf("semverRe.MatchString(%q) = %v, want %v", tt.version, !tt.valid, tt.valid)
			}
		})
	}
}
