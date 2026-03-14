// Package install implements the mint install command logic.
package install

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DefaultRegistryURL is the default registry API base URL.
const DefaultRegistryURL = "https://api.mintmcp.com/v1"

// Options controls the install behavior.
type Options struct {
	Name        string // server name, optionally with @version
	RegistryURL string // registry base URL
	InstallDir  string // directory to install into (default: ~/.mint/servers)
}

// ParseNameVersion parses "name" or "name@version" into separate components.
func ParseNameVersion(s string) (name, version string) {
	if i := strings.LastIndex(s, "@"); i > 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// installDir returns the default install directory.
func defaultInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mint", "servers"), nil
}

// Install downloads and extracts an MCP server from the registry.
func Install(opts Options) (string, error) {
	if opts.RegistryURL == "" {
		opts.RegistryURL = DefaultRegistryURL
	}
	if opts.InstallDir == "" {
		dir, err := defaultInstallDir()
		if err != nil {
			return "", err
		}
		opts.InstallDir = dir
	}

	name, version := ParseNameVersion(opts.Name)

	// First, look up the server by name to get its ID.
	serverID, err := resolveServerID(opts.RegistryURL, name)
	if err != nil {
		return "", err
	}

	// Download the artifact.
	downloadURL := fmt.Sprintf("%s/servers/%s/download", strings.TrimRight(opts.RegistryURL, "/"), serverID)
	if version != "" {
		downloadURL += "?version=" + version
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return "", fmt.Errorf("download failed (%d): %s", resp.StatusCode, errResp.Error)
		}
		return "", fmt.Errorf("download failed (%d): %s", resp.StatusCode, string(body))
	}

	// Extract to install directory.
	destDir := filepath.Join(opts.InstallDir, name)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("create install directory: %w", err)
	}

	if err := extractTarGz(resp.Body, destDir); err != nil {
		return "", fmt.Errorf("extract archive: %w", err)
	}

	return destDir, nil
}

// resolveServerID looks up a server by name and returns its ID.
func resolveServerID(registryURL, name string) (string, error) {
	url := fmt.Sprintf("%s/servers?q=%s&page_size=1", strings.TrimRight(registryURL, "/"), name)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search failed (%d)", resp.StatusCode)
	}

	var result struct {
		Servers []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"servers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse search response: %w", err)
	}

	for _, s := range result.Servers {
		if s.Name == name {
			return s.ID, nil
		}
	}
	return "", fmt.Errorf("server %q not found in registry", name)
}

// extractTarGz extracts a gzipped tarball from r into destDir.
func extractTarGz(r io.Reader, destDir string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip the top-level directory from the tarball path.
		name := hdr.Name
		if i := strings.Index(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if name == "" {
			continue
		}

		target := filepath.Join(destDir, name)

		// Prevent path traversal.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("invalid tar entry path: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}
