// Package publish implements the mint publish command logic.
package publish

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DefaultRegistryURL is the default registry API base URL.
const DefaultRegistryURL = "https://api.mint.sire.run/v1"

// Manifest represents the mint.json file in a project.
type Manifest struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Description    string `json:"description"`
	Category       string `json:"category,omitempty"`
	OpenAPISpecURL string `json:"openapi_spec_url,omitempty"`
	Changelog      string `json:"changelog,omitempty"`
}

// Options controls the publish behavior.
type Options struct {
	Dir         string // project directory (default: cwd)
	RegistryURL string // registry base URL
	Token       string // API token
	DryRun      bool   // validate only, don't upload
}

// excludedDirs are directories excluded from the tarball.
var excludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".DS_Store":    true,
	"__pycache__":  true,
	".env":         true,
	".venv":        true,
	"vendor":       true,
}

// ReadManifest reads and validates the mint.json manifest from dir.
func ReadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "mint.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("mint.json not found in %s", dir)
		}
		return nil, fmt.Errorf("read mint.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse mint.json: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Validate checks that required manifest fields are present.
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return errors.New("mint.json: name is required")
	}
	if m.Version == "" {
		return errors.New("mint.json: version is required")
	}
	if m.Description == "" {
		return errors.New("mint.json: description is required")
	}
	return nil
}

// PackageTarball creates a gzipped tarball of the project directory,
// excluding common directories like .git and node_modules.
func PackageTarball(dir string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	baseDir := filepath.Base(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from the project directory.
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Skip excluded directories.
		if info.IsDir() && excludedDirs[info.Name()] {
			return filepath.SkipDir
		}

		// Skip excluded files.
		if !info.IsDir() && excludedDirs[info.Name()] {
			return nil
		}

		// Create tar header.
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.Join(baseDir, rel)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create tarball: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}

// PublishResponse is the response from the registry publish endpoint.
type PublishResponse struct {
	ServerID  string `json:"server_id"`
	VersionID string `json:"version_id"`
	Version   string `json:"version"`
	Checksum  string `json:"checksum"`
}

// Upload packages and uploads the project to the registry.
// It returns the publish response or an error.
func Upload(opts Options) (*PublishResponse, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.RegistryURL == "" {
		opts.RegistryURL = DefaultRegistryURL
	}

	manifest, err := ReadManifest(opts.Dir)
	if err != nil {
		return nil, err
	}

	if opts.DryRun {
		return &PublishResponse{Version: manifest.Version}, nil
	}

	tarball, err := PackageTarball(opts.Dir)
	if err != nil {
		return nil, err
	}

	// Build multipart request.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Write metadata field.
	metadata, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if err := writer.WriteField("metadata", string(metadata)); err != nil {
		return nil, err
	}

	// Write artifact file.
	part, err := writer.CreateFormFile("artifact", manifest.Name+"-"+manifest.Version+".tar.gz")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, tarball); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	url := strings.TrimRight(opts.RegistryURL, "/") + "/publish"
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+opts.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("publish failed (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("publish failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result PublishResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}
