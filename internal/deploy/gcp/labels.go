// Package gcp provides utilities for deploying MCP servers to Google Cloud Run.
package gcp

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// DeployLabels constructs labels for a Cloud Run service deployment.
// Labels provide audit metadata for SOC2 compliance.
func DeployLabels(mintVersion, specHash, commitSHA string) map[string]string {
	username := "unknown"
	if u, err := user.Current(); err == nil && u.Username != "" {
		username = SanitizeLabel(u.Username)
	}

	now := time.Now().UTC().Format("2006-01-02t15-04-05z")

	sh := specHash
	if len(sh) > 12 {
		sh = sh[:12]
	}

	cs := commitSHA
	if len(cs) > 12 {
		cs = cs[:12]
	}
	if cs == "" {
		cs = "unknown"
	}

	return map[string]string{
		"mint-version": SanitizeLabel(mintVersion),
		"spec-hash":    sh,
		"commit-sha":   cs,
		"deployed-by":  username,
		"deployed-at":  now,
	}
}

// SanitizeLabel sanitizes a string for use as a GCP label value.
// GCP labels must be lowercase, max 63 characters, contain only
// letters, numbers, hyphens, and underscores, and must start with
// a letter or number.
func SanitizeLabel(s string) string {
	s = strings.ToLower(s)

	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// Strip leading characters that are not letters or digits.
	for len(s) > 0 && !unicode.IsLetter(rune(s[0])) && !unicode.IsDigit(rune(s[0])) {
		s = s[1:]
	}

	if len(s) > 63 {
		s = s[:63]
	}

	return s
}

// SpecHash computes a SHA256 hash of all .go, .mod, and Dockerfile files
// in the given source directory. It returns the first 12 hex characters.
func SpecHash(sourceDir string) string {
	h := sha256.New()

	_ = filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		ext := filepath.Ext(name)
		include := ext == ".go" || ext == ".mod" || name == "Dockerfile"
		if !include {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Include relative path so file renames change the hash.
		rel, _ := filepath.Rel(sourceDir, path)
		_, _ = fmt.Fprintf(h, "%s\n", rel)
		h.Write(data)

		return nil
	})

	sum := fmt.Sprintf("%x", h.Sum(nil))
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}
