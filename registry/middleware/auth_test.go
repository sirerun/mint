package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/sirerun/mint/registry/db"
	"github.com/sirerun/mint/registry/model"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestAuth_Required_ValidKey(t *testing.T) {
	store := setupTestDB(t)
	apiKey := "test-api-key"
	pub := &model.Publisher{
		ID:           "pub-1",
		GithubHandle: "testuser",
		APIKeyHash:   db.HashAPIKey(apiKey),
	}
	store.CreatePublisher(pub)

	var gotPublisher *model.Publisher
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPublisher = PublisherFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(store, true)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotPublisher == nil {
		t.Fatal("expected publisher in context")
	}
	if gotPublisher.ID != pub.ID {
		t.Errorf("publisher ID = %q, want %q", gotPublisher.ID, pub.ID)
	}
}

func TestAuth_Required_NoKey(t *testing.T) {
	store := setupTestDB(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(store, true)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_Required_InvalidKey(t *testing.T) {
	store := setupTestDB(t)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(store, true)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuth_Optional_NoKey(t *testing.T) {
	store := setupTestDB(t)
	var gotPublisher *model.Publisher
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPublisher = PublisherFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(store, false)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotPublisher != nil {
		t.Error("expected nil publisher for unauthenticated request")
	}
}

func TestAuth_Optional_ValidKey(t *testing.T) {
	store := setupTestDB(t)
	apiKey := "test-key"
	pub := &model.Publisher{
		ID:           "pub-1",
		GithubHandle: "testuser",
		APIKeyHash:   db.HashAPIKey(apiKey),
	}
	store.CreatePublisher(pub)

	var gotPublisher *model.Publisher
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPublisher = PublisherFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Auth(store, false)(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if gotPublisher == nil || gotPublisher.ID != pub.ID {
		t.Error("expected publisher in context for valid key")
	}
}

func TestPublisherFromContext_NilContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	p := PublisherFromContext(req.Context())
	if p != nil {
		t.Error("expected nil publisher from empty context")
	}
}
