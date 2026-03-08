package gcp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDeployLabelsReturnsAllExpectedKeys(t *testing.T) {
	labels := DeployLabels("v1.2.3", "abcdef123456789", "aabbccdd1122")

	expectedKeys := []string{
		"mint-version",
		"spec-hash",
		"commit-sha",
		"deployed-by",
		"deployed-at",
	}

	for _, key := range expectedKeys {
		if _, ok := labels[key]; !ok {
			t.Errorf("missing expected label key %q", key)
		}
	}

	if got := labels["mint-version"]; got != "v123" {
		t.Errorf("mint-version = %q, want %q", got, "v123")
	}

	if got := labels["spec-hash"]; got != "abcdef123456" {
		t.Errorf("spec-hash = %q, want %q", got, "abcdef123456")
	}

	if got := labels["commit-sha"]; got != "aabbccdd1122" {
		t.Errorf("commit-sha = %q, want %q", got, "aabbccdd1122")
	}

	if got := labels["deployed-by"]; got == "" {
		t.Error("deployed-by should not be empty")
	}
}

func TestDeployLabelsCommitSHAUnknownWhenEmpty(t *testing.T) {
	labels := DeployLabels("v1.0.0", "abc", "")
	if got := labels["commit-sha"]; got != "unknown" {
		t.Errorf("commit-sha = %q, want %q", got, "unknown")
	}
}

func TestDeployLabelsDeployedAtFormat(t *testing.T) {
	labels := DeployLabels("v1.0.0", "abc", "def")

	// Format: 2006-01-02t15-04-05z
	pattern := `^\d{4}-\d{2}-\d{2}t\d{2}-\d{2}-\d{2}z$`
	re := regexp.MustCompile(pattern)
	if !re.MatchString(labels["deployed-at"]) {
		t.Errorf("deployed-at = %q, does not match pattern %s", labels["deployed-at"], pattern)
	}
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase passthrough",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "uppercase converted",
			input: "Hello-World",
			want:  "hello-world",
		},
		{
			name:  "special chars removed",
			input: "hello@world!",
			want:  "helloworld",
		},
		{
			name:  "too long truncated",
			input: strings.Repeat("a", 100),
			want:  strings.Repeat("a", 63),
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "leading hyphens stripped",
			input: "--hello",
			want:  "hello",
		},
		{
			name:  "underscores preserved",
			input: "hello_world",
			want:  "hello_world",
		},
		{
			name:  "digits allowed",
			input: "v1.2.3",
			want:  "v123",
		},
		{
			name:  "starts with digit",
			input: "123abc",
			want:  "123abc",
		},
		{
			name:  "domain style username",
			input: "DOMAIN\\user",
			want:  "domainuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLabel(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSpecHash(t *testing.T) {
	dir := t.TempDir()

	// Create test files.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM golang:1.25"), 0o644); err != nil {
		t.Fatal(err)
	}
	// This file should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash := SpecHash(dir)

	if len(hash) != 12 {
		t.Errorf("SpecHash length = %d, want 12", len(hash))
	}

	// Hash should be hex characters only.
	for _, c := range hash {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("SpecHash contains non-hex character %q", string(c))
		}
	}

	// Hash should be deterministic.
	hash2 := SpecHash(dir)
	if hash != hash2 {
		t.Errorf("SpecHash not deterministic: %q != %q", hash, hash2)
	}

	// Changing a file should change the hash.
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n// changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash3 := SpecHash(dir)
	if hash3 == hash {
		t.Error("SpecHash should change when file content changes")
	}
}

func TestSpecHashEmptyDir(t *testing.T) {
	dir := t.TempDir()
	hash := SpecHash(dir)

	if len(hash) != 12 {
		t.Errorf("SpecHash length = %d, want 12", len(hash))
	}
}
