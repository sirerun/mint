package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testIndex() *RegistryIndex {
	return &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "github", Description: "GitHub API v3", Tags: []string{"scm"}},
			{Name: "stripe", Description: "Stripe payments", Tags: []string{"payments"}},
		},
	}
}

func serveIndex(t *testing.T, index *RegistryIndex) *httptest.Server {
	t.Helper()
	data, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshal test index: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
}

func TestFetchIndex_Success(t *testing.T) {
	idx := testIndex()
	srv := serveIndex(t, idx)
	defer srv.Close()

	got, err := FetchIndex(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchIndex: %v", err)
	}
	if got.Version != idx.Version {
		t.Errorf("Version = %d, want %d", got.Version, idx.Version)
	}
	if len(got.Entries) != len(idx.Entries) {
		t.Errorf("Entries len = %d, want %d", len(got.Entries), len(idx.Entries))
	}
}

func TestFetchIndex_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := FetchIndex(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestFetchIndex_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := FetchIndex(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchIndex_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := FetchIndex(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestLoadCachedIndex_Success(t *testing.T) {
	dir := t.TempDir()
	idx := testIndex()
	data, _ := json.Marshal(idx)
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	got, modTime, err := LoadCachedIndex(dir)
	if err != nil {
		t.Fatalf("LoadCachedIndex: %v", err)
	}
	if got.Version != idx.Version {
		t.Errorf("Version = %d, want %d", got.Version, idx.Version)
	}
	if modTime.IsZero() {
		t.Error("modTime is zero")
	}
}

func TestLoadCachedIndex_Missing(t *testing.T) {
	dir := t.TempDir()
	_, _, err := LoadCachedIndex(dir)
	if err == nil {
		t.Fatal("expected error for missing cache")
	}
}

func TestLoadCachedIndex_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), []byte("bad"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, _, err := LoadCachedIndex(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON cache")
	}
}

func TestSaveCache_Success(t *testing.T) {
	dir := t.TempDir()
	idx := testIndex()

	if err := SaveCache(dir, idx); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, cacheFileName))
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}

	var got RegistryIndex
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Version != idx.Version {
		t.Errorf("Version = %d, want %d", got.Version, idx.Version)
	}
}

func TestSaveCache_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	idx := testIndex()

	if err := SaveCache(dir, idx); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, cacheFileName)); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}
}

func TestGetIndex_FreshFetch(t *testing.T) {
	idx := testIndex()
	srv := serveIndex(t, idx)
	defer srv.Close()

	dir := t.TempDir()
	got, err := GetIndex(context.Background(), IndexOptions{
		IndexURL: srv.URL,
		CacheDir: dir,
	})
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != idx.Version {
		t.Errorf("Version = %d, want %d", got.Version, idx.Version)
	}

	// Verify cache was written.
	if _, err := os.Stat(filepath.Join(dir, cacheFileName)); err != nil {
		t.Errorf("cache not written: %v", err)
	}
}

func TestGetIndex_CacheHit(t *testing.T) {
	dir := t.TempDir()
	idx := testIndex()
	if err := SaveCache(dir, idx); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	// Server that would fail if called.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called for fresh cache")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	got, err := GetIndex(context.Background(), IndexOptions{
		IndexURL: srv.URL,
		CacheDir: dir,
	})
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != idx.Version {
		t.Errorf("Version = %d, want %d", got.Version, idx.Version)
	}
}

func TestGetIndex_ExpiredCache_FetchSucceeds(t *testing.T) {
	dir := t.TempDir()
	oldIdx := &RegistryIndex{Version: 1, Entries: []RegistryEntry{{Name: "old"}}}
	data, _ := json.Marshal(oldIdx)
	p := filepath.Join(dir, cacheFileName)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Set mod time to 2 hours ago.
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	newIdx := &RegistryIndex{Version: 2, Entries: []RegistryEntry{{Name: "new"}}}
	srv := serveIndex(t, newIdx)
	defer srv.Close()

	got, err := GetIndex(context.Background(), IndexOptions{
		IndexURL: srv.URL,
		CacheDir: dir,
	})
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2", got.Version)
	}
}

func TestGetIndex_OfflineFallback(t *testing.T) {
	dir := t.TempDir()
	idx := testIndex()
	data, _ := json.Marshal(idx)
	p := filepath.Join(dir, cacheFileName)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Make cache stale.
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	// Server returns error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	got, err := GetIndex(context.Background(), IndexOptions{
		IndexURL: srv.URL,
		CacheDir: dir,
	})
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != idx.Version {
		t.Errorf("Version = %d, want %d (stale cache fallback)", got.Version, idx.Version)
	}
}

func TestGetIndex_NoCacheNoNetwork(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := GetIndex(context.Background(), IndexOptions{
		IndexURL: srv.URL,
		CacheDir: dir,
	})
	if err == nil {
		t.Fatal("expected error when no cache and network fails")
	}
}

func TestGetIndex_ForceFresh(t *testing.T) {
	dir := t.TempDir()
	// Write fresh cache that would be a hit.
	cachedIdx := &RegistryIndex{Version: 1, Entries: []RegistryEntry{{Name: "cached"}}}
	if err := SaveCache(dir, cachedIdx); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	freshIdx := &RegistryIndex{Version: 2, Entries: []RegistryEntry{{Name: "fresh"}}}
	srv := serveIndex(t, freshIdx)
	defer srv.Close()

	got, err := GetIndex(context.Background(), IndexOptions{
		IndexURL:   srv.URL,
		CacheDir:   dir,
		ForceFresh: true,
	})
	if err != nil {
		t.Fatalf("GetIndex: %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2 (force fresh)", got.Version)
	}
}

func TestGetIndex_ForceFresh_NoFallback(t *testing.T) {
	dir := t.TempDir()
	// Write stale cache.
	cachedIdx := &RegistryIndex{Version: 1}
	if err := SaveCache(dir, cachedIdx); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := GetIndex(context.Background(), IndexOptions{
		IndexURL:   srv.URL,
		CacheDir:   dir,
		ForceFresh: true,
	})
	if err == nil {
		t.Fatal("expected error: ForceFresh should not fall back to cache")
	}
}
