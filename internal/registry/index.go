package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DefaultIndexURL is the canonical location for the MCP server registry index.
const DefaultIndexURL = "https://raw.githubusercontent.com/sirerun/mcp-registry/main/registry.json"

const cacheFileName = "registry.json"
const cacheTTL = 1 * time.Hour

// IndexOptions configures how GetIndex resolves the registry index.
type IndexOptions struct {
	IndexURL   string
	CacheDir   string
	ForceFresh bool
}

// FetchIndex fetches the registry index from the given URL.
func FetchIndex(ctx context.Context, indexURL string) (*RegistryIndex, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching index: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return &index, nil
}

// LoadCachedIndex loads the registry index from the cache directory.
// It returns the index, the modification time of the cache file, and any error.
func LoadCachedIndex(cacheDir string) (*RegistryIndex, time.Time, error) {
	p := filepath.Join(cacheDir, cacheFileName)
	info, err := os.Stat(p)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat cache: %w", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("reading cache: %w", err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, time.Time{}, fmt.Errorf("parsing cache: %w", err)
	}
	return &index, info.ModTime(), nil
}

// SaveCache writes the registry index to the cache directory.
func SaveCache(cacheDir string, index *RegistryIndex) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	data, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	p := filepath.Join(cacheDir, cacheFileName)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}
	return nil
}

func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "mint")
}

// GetIndex is the main entry point for obtaining the registry index.
// It checks cache freshness, fetches from the network when needed,
// and falls back to stale cache when the network is unavailable.
func GetIndex(ctx context.Context, opts IndexOptions) (*RegistryIndex, error) {
	if opts.IndexURL == "" {
		opts.IndexURL = DefaultIndexURL
	}
	if opts.CacheDir == "" {
		opts.CacheDir = defaultCacheDir()
	}

	// If not forcing fresh, check cache first.
	if !opts.ForceFresh {
		cached, modTime, err := LoadCachedIndex(opts.CacheDir)
		if err == nil && time.Since(modTime) < cacheTTL {
			return cached, nil
		}
	}

	// Fetch fresh index from network.
	index, fetchErr := FetchIndex(ctx, opts.IndexURL)
	if fetchErr == nil {
		// Save to cache (best effort).
		_ = SaveCache(opts.CacheDir, index)
		return index, nil
	}

	// Fetch failed -- fall back to stale cache if available.
	if !opts.ForceFresh {
		cached, _, err := LoadCachedIndex(opts.CacheDir)
		if err == nil {
			fmt.Fprintf(os.Stderr, "Warning: using stale registry cache (fetch failed: %v)\n", fetchErr)
			return cached, nil
		}
	}

	return nil, fmt.Errorf("registry unavailable: %w", fetchErr)
}
