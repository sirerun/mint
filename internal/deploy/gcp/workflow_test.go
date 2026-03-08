package gcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validConfig() WorkflowConfig {
	return WorkflowConfig{
		ProjectID:                "my-project",
		Region:                   "us-central1",
		ServiceName:              "my-service",
		SourceDir:                "server",
		WorkloadIdentityProvider: "projects/123/locations/global/workloadIdentityPools/pool/providers/provider",
		ServiceAccountEmail:      "sa@my-project.iam.gserviceaccount.com",
	}
}

func TestGenerateWorkflow_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	config := validConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	if result.Content == "" {
		t.Fatal("expected non-empty content")
	}
	if result.FilePath == "" {
		t.Fatal("expected non-empty file path")
	}

	// Verify the file was actually written and matches content.
	data, err := os.ReadFile(result.FilePath)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(data) != result.Content {
		t.Error("file content does not match result content")
	}
}

func TestGenerateWorkflow_ContainsConfigValues(t *testing.T) {
	dir := t.TempDir()
	config := validConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	checks := []struct {
		label string
		want  string
	}{
		{"project ID", config.ProjectID},
		{"region", config.Region},
		{"service name", config.ServiceName},
		{"source dir", config.SourceDir},
		{"workload identity provider", config.WorkloadIdentityProvider},
		{"service account email", config.ServiceAccountEmail},
	}

	for _, c := range checks {
		if !strings.Contains(result.Content, c.want) {
			t.Errorf("content missing %s (%q)", c.label, c.want)
		}
	}
}

func TestGenerateWorkflow_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	config := validConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	expected := filepath.Join(dir, ".github", "workflows", "deploy-gcp.yml")
	if result.FilePath != expected {
		t.Errorf("FilePath = %q, want %q", result.FilePath, expected)
	}

	info, err := os.Stat(filepath.Join(dir, ".github", "workflows"))
	if err != nil {
		t.Fatalf("stat .github/workflows: %v", err)
	}
	if !info.IsDir() {
		t.Error(".github/workflows is not a directory")
	}
}

func TestGenerateWorkflow_MissingRequiredConfig(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*WorkflowConfig)
		want   string
	}{
		{
			name:   "missing ProjectID",
			modify: func(c *WorkflowConfig) { c.ProjectID = "" },
			want:   "projectID",
		},
		{
			name:   "missing Region",
			modify: func(c *WorkflowConfig) { c.Region = "" },
			want:   "region",
		},
		{
			name:   "missing ServiceName",
			modify: func(c *WorkflowConfig) { c.ServiceName = "" },
			want:   "serviceName",
		},
		{
			name:   "missing SourceDir",
			modify: func(c *WorkflowConfig) { c.SourceDir = "" },
			want:   "sourceDir",
		},
		{
			name:   "missing WorkloadIdentityProvider",
			modify: func(c *WorkflowConfig) { c.WorkloadIdentityProvider = "" },
			want:   "workloadIdentityProvider",
		},
		{
			name:   "missing ServiceAccountEmail",
			modify: func(c *WorkflowConfig) { c.ServiceAccountEmail = "" },
			want:   "serviceAccountEmail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validConfig()
			tt.modify(&config)

			_, err := GenerateWorkflow(config, t.TempDir())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q should mention %q", err, tt.want)
			}
		})
	}
}
