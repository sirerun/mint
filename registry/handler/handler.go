// Package handler implements the HTTP handlers for the Mint registry API.
package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirerun/mint/registry/db"
	"github.com/sirerun/mint/registry/middleware"
	"github.com/sirerun/mint/registry/model"
)

var semverRe = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([\da-zA-Z\-]+(?:\.[\da-zA-Z\-]+)*))?(?:\+([\da-zA-Z\-]+(?:\.[\da-zA-Z\-]+)*))?$`)

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9\-]{1,62}[a-z0-9]$`)

// Handler holds dependencies for all registry endpoints.
type Handler struct {
	DB          *db.DB
	ArtifactDir string // directory for storing uploaded artifacts
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg})
}

// Publish handles POST /publish.
func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	publisher := middleware.PublisherFromContext(r.Context())
	if publisher == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Parse multipart form (max 50MB).
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
		return
	}

	// Parse metadata.
	metadataStr := r.FormValue("metadata")
	if metadataStr == "" {
		writeError(w, http.StatusBadRequest, "metadata field is required")
		return
	}
	var req model.PublishRequest
	if err := json.Unmarshal([]byte(metadataStr), &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid metadata JSON: "+err.Error())
		return
	}

	// Validate fields.
	if !nameRe.MatchString(req.Name) {
		writeError(w, http.StatusBadRequest, "name must be 3-64 lowercase alphanumeric characters and hyphens, starting with a letter")
		return
	}
	if !semverRe.MatchString(req.Version) {
		writeError(w, http.StatusBadRequest, "version must be valid semver (e.g., 1.0.0 or v1.0.0)")
		return
	}
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	// Get artifact file.
	file, _, err := r.FormFile("artifact")
	if err != nil {
		writeError(w, http.StatusBadRequest, "artifact file is required")
		return
	}
	defer file.Close()

	// Generate IDs.
	serverID := generateID()
	versionID := generateID()

	// Check if server name already exists.
	existing, err := h.DB.GetServerByName(req.Name)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if existing != nil {
		// Server exists - verify ownership and add new version.
		if existing.PublisherID != publisher.ID {
			writeError(w, http.StatusForbidden, "server name is owned by another publisher")
			return
		}
		serverID = existing.ID

		// Check version doesn't already exist.
		_, err := h.DB.GetVersion(serverID, req.Version)
		if err == nil {
			writeError(w, http.StatusConflict, "version already exists")
			return
		}
		if !errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}

	// Store artifact.
	artifactDir := filepath.Join(h.ArtifactDir, serverID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create artifact directory")
		return
	}
	artifactPath := filepath.Join(artifactDir, req.Version+".tar.gz")
	dst, err := os.Create(artifactPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create artifact file")
		return
	}
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(dst, hasher), file)
	dst.Close()
	if err != nil {
		_ = os.Remove(artifactPath)
		writeError(w, http.StatusInternalServerError, "failed to write artifact")
		return
	}
	_ = written
	checksum := hex.EncodeToString(hasher.Sum(nil))

	// Create or update server record.
	if existing == nil {
		srv := &model.Server{
			ID:             serverID,
			Name:           req.Name,
			Description:    req.Description,
			LatestVersion:  req.Version,
			OpenAPISpecURL: req.OpenAPISpecURL,
			PublisherID:    publisher.ID,
			Category:       req.Category,
		}
		if err := h.DB.CreateServer(srv); err != nil {
			_ = os.Remove(artifactPath)
			writeError(w, http.StatusInternalServerError, "failed to create server record: "+err.Error())
			return
		}
	} else {
		existing.Description = req.Description
		existing.LatestVersion = req.Version
		if req.OpenAPISpecURL != "" {
			existing.OpenAPISpecURL = req.OpenAPISpecURL
		}
		if req.Category != "" {
			existing.Category = req.Category
		}
		if err := h.DB.UpdateServer(existing); err != nil {
			_ = os.Remove(artifactPath)
			writeError(w, http.StatusInternalServerError, "failed to update server record")
			return
		}
	}

	// Create version record.
	ver := &model.Version{
		ID:           versionID,
		ServerID:     serverID,
		Version:      req.Version,
		ArtifactPath: artifactPath,
		Checksum:     checksum,
		Changelog:    req.Changelog,
	}
	if err := h.DB.CreateVersion(ver); err != nil {
		_ = os.Remove(artifactPath)
		writeError(w, http.StatusInternalServerError, "failed to create version record: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"server_id":  serverID,
		"version_id": versionID,
		"version":    req.Version,
		"checksum":   checksum,
	})
}

// ListServers handles GET /servers.
func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	sort := r.URL.Query().Get("sort")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	result, err := h.DB.SearchServers(q, category, sort, page, pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetServer handles GET /servers/{id}.
func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := extractPathParam(r.URL.Path, "/api/v1/servers/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "server id is required")
		return
	}
	// Strip any trailing path segments (for /download, /star, /stars).
	if idx := strings.Index(id, "/"); idx != -1 {
		id = id[:idx]
	}

	srv, err := h.DB.GetServerByID(id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	versions, err := h.DB.ListVersions(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list versions")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"server":   srv,
		"versions": versions,
	})
}

// Download handles GET /servers/{id}/download.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract server ID from path.
	path := r.URL.Path
	id := extractServerIDFromSubpath(path, "download")
	if id == "" {
		writeError(w, http.StatusBadRequest, "server id is required")
		return
	}

	// Verify server exists.
	srv, err := h.DB.GetServerByID(id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Get requested version or latest.
	versionStr := r.URL.Query().Get("version")
	var ver *model.Version
	if versionStr == "" || versionStr == "latest" {
		ver, err = h.DB.GetLatestVersion(srv.ID)
	} else {
		ver, err = h.DB.GetVersion(srv.ID, versionStr)
	}
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Increment download counter.
	_ = h.DB.IncrementDownloads(srv.ID)

	// Serve the file.
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.tar.gz", srv.Name, ver.Version))
	w.Header().Set("X-Checksum-SHA256", ver.Checksum)
	http.ServeFile(w, r, ver.ArtifactPath)
}

// StarToggle handles POST /servers/{id}/star.
func (h *Handler) StarToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	publisher := middleware.PublisherFromContext(r.Context())
	if publisher == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := extractServerIDFromSubpath(r.URL.Path, "star")
	if id == "" {
		writeError(w, http.StatusBadRequest, "server id is required")
		return
	}

	// Verify server exists.
	_, err := h.DB.GetServerByID(id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	added, err := h.DB.ToggleStar(publisher.ID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to toggle star: "+err.Error())
		return
	}

	count, _ := h.DB.GetStarCount(id)

	writeJSON(w, http.StatusOK, map[string]any{
		"starred": added,
		"stars":   count,
	})
}

// GetStars handles GET /servers/{id}/stars.
func (h *Handler) GetStars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := extractServerIDFromSubpath(r.URL.Path, "stars")
	if id == "" {
		writeError(w, http.StatusBadRequest, "server id is required")
		return
	}

	count, err := h.DB.GetStarCount(id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"stars": count})
}

// RegisterPublisher handles POST /publishers/register.
func (h *Handler) RegisterPublisher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		GithubHandle string `json:"github_handle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.GithubHandle == "" {
		writeError(w, http.StatusBadRequest, "github_handle is required")
		return
	}

	// Generate API key.
	apiKey := generateAPIKey()
	hash := db.HashAPIKey(apiKey)

	pub := &model.Publisher{
		ID:           generateID(),
		GithubHandle: req.GithubHandle,
		APIKeyHash:   hash,
	}
	if err := h.DB.CreatePublisher(pub); err != nil {
		if errors.Is(err, db.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "publisher already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to register publisher: "+err.Error())
		return
	}

	// Return the API key once; it can never be retrieved again.
	writeJSON(w, http.StatusCreated, map[string]string{
		"publisher_id": pub.ID,
		"api_key":      apiKey,
		"message":      "Save this API key securely. It cannot be retrieved again.",
	})
}

// VerifyPublisher handles POST /publishers/{id}/verify.
func (h *Handler) VerifyPublisher(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// For MVP, this is a simplified endpoint. In production, this would
	// verify domain ownership or GitHub org membership.
	publisher := middleware.PublisherFromContext(r.Context())
	if publisher == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id := extractPathParam(r.URL.Path, "/api/v1/publishers/")
	if idx := strings.Index(id, "/"); idx != -1 {
		id = id[:idx]
	}

	if publisher.ID != id {
		writeError(w, http.StatusForbidden, "can only verify your own publisher account")
		return
	}

	if err := h.DB.SetPublisherVerified(id, true); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "publisher not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verified": true,
	})
}

func extractPathParam(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.TrimPrefix(path, prefix)
}

// extractServerIDFromSubpath extracts the server ID from paths like
// /api/v1/servers/{id}/download or /api/v1/servers/{id}/star.
func extractServerIDFromSubpath(path, suffix string) string {
	prefix := "/api/v1/servers/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	// rest should be "{id}/suffix"
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || parts[1] != suffix {
		return ""
	}
	return parts[0]
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateAPIKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return "mint_" + hex.EncodeToString(b)
}
