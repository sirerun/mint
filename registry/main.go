// Package main is the entrypoint for the Mint registry server.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/sirerun/mint/registry/db"
	"github.com/sirerun/mint/registry/handler"
	"github.com/sirerun/mint/registry/middleware"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "registry.db", "SQLite database path")
	artifactDir := flag.String("artifacts", "artifacts", "artifact storage directory")
	flag.Parse()

	if envAddr := os.Getenv("REGISTRY_ADDR"); envAddr != "" {
		*addr = envAddr
	}
	if envDB := os.Getenv("REGISTRY_DB"); envDB != "" {
		*dbPath = envDB
	}
	if envArt := os.Getenv("REGISTRY_ARTIFACTS"); envArt != "" {
		*artifactDir = envArt
	}

	if err := os.MkdirAll(*artifactDir, 0o755); err != nil {
		log.Fatalf("create artifact directory: %v", err)
	}

	store, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close() //nolint:errcheck

	h := &handler.Handler{
		DB:          store,
		ArtifactDir: *artifactDir,
	}

	mux := buildMux(h, store)

	log.Printf("Mint registry listening on %s", *addr)
	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("server error: %v", err)
	}
}

func buildMux(h *handler.Handler, store *db.DB) http.Handler {
	mux := http.NewServeMux()

	// Rate limiters.
	publishLimiter := middleware.NewRateLimiter(10, time.Hour)
	searchLimiter := middleware.NewRateLimiter(100, time.Minute)

	authRequired := middleware.Auth(store, true)
	authOptional := middleware.Auth(store, false)
	publishRateLimit := middleware.RateLimit(publishLimiter, middleware.PublisherKeyFunc)
	searchRateLimit := middleware.RateLimit(searchLimiter, middleware.IPKeyFunc)

	// Public endpoints.
	mux.Handle("/api/v1/servers", chain(http.HandlerFunc(h.ListServers), searchRateLimit, authOptional))
	mux.Handle("/api/v1/publishers/register", http.HandlerFunc(h.RegisterPublisher))

	// Authenticated endpoints.
	mux.Handle("/api/v1/publish", chain(http.HandlerFunc(h.Publish), publishRateLimit, authRequired))

	// Server sub-routes need custom routing since http.ServeMux pattern matching
	// in older Go versions doesn't support path params.
	mux.Handle("/api/v1/servers/", chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeServerSubpath(h, w, r)
	}), authOptional))

	// Publisher sub-routes.
	mux.Handle("/api/v1/publishers/", chain(http.HandlerFunc(h.VerifyPublisher), authRequired))

	// Health check.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	return mux
}

func routeServerSubpath(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /api/v1/servers/{id}/download
	if matched, _ := matchSubpath(path, "download"); matched {
		h.Download(w, r)
		return
	}
	// /api/v1/servers/{id}/star
	if matched, _ := matchSubpath(path, "star"); matched {
		h.StarToggle(w, r)
		return
	}
	// /api/v1/servers/{id}/stars
	if matched, _ := matchSubpath(path, "stars"); matched {
		h.GetStars(w, r)
		return
	}
	// /api/v1/servers/{id}
	h.GetServer(w, r)
}

func matchSubpath(path, suffix string) (bool, string) {
	prefix := "/api/v1/servers/"
	rest := path[len(prefix):]
	parts := splitPath(rest)
	if len(parts) == 2 && parts[1] == suffix {
		return true, parts[0]
	}
	return false, ""
}

func splitPath(s string) []string {
	var parts []string
	for _, p := range splitOn(s, '/') {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitOn(s string, sep byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// chain applies middleware in reverse order so the first middleware
// in the argument list is the outermost wrapper.
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
