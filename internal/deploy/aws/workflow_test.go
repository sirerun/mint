package aws

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		Region:      "us-east-1",
		ServiceName: "my-service",
		SourceDir:   "server",
		RoleARN:     "arn:aws:iam::123456789012:role/mint-github-deploy-my-service",
		AccountID:   "123456789012",
	}
}

func TestGenerateWorkflow_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	config := validWorkflowConfig()

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
	config := validWorkflowConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	checks := []struct {
		label string
		want  string
	}{
		{"region", config.Region},
		{"service name", config.ServiceName},
		{"source dir", config.SourceDir},
		{"role ARN", config.RoleARN},
	}

	for _, c := range checks {
		if !strings.Contains(result.Content, c.want) {
			t.Errorf("content missing %s (%q)", c.label, c.want)
		}
	}
}

func TestGenerateWorkflow_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	config := validWorkflowConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	expected := filepath.Join(dir, ".github", "workflows", "deploy-aws.yml")
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

func TestGenerateWorkflow_TemplateParseError(t *testing.T) {
	original := workflowTemplateStr
	t.Cleanup(func() { workflowTemplateStr = original })

	workflowTemplateStr = "{{.Invalid"
	config := validWorkflowConfig()

	_, err := GenerateWorkflow(config, t.TempDir())
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "parsing workflow template") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "parsing workflow template")
	}
}

func TestGenerateWorkflow_TemplateExecuteError(t *testing.T) {
	original := workflowTemplateStr
	t.Cleanup(func() { workflowTemplateStr = original })

	// A template that calls a missing function will fail on Execute.
	workflowTemplateStr = "{{call .Missing}}"
	config := validWorkflowConfig()

	_, err := GenerateWorkflow(config, t.TempDir())
	if err == nil {
		t.Fatal("expected error for template execution failure")
	}
	if !strings.Contains(err.Error(), "executing workflow template") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "executing workflow template")
	}
}

func TestGenerateWorkflow_InvalidOutputDir(t *testing.T) {
	config := validWorkflowConfig()

	// Use /dev/null as outputDir — it exists as a file, so MkdirAll will fail
	// trying to create a subdirectory under it.
	_, err := GenerateWorkflow(config, "/dev/null")
	if err == nil {
		t.Fatal("expected error for invalid output directory")
	}
	if !strings.Contains(err.Error(), "creating workflow directory") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "creating workflow directory")
	}
}

func TestGenerateWorkflow_WriteFileError(t *testing.T) {
	config := validWorkflowConfig()

	// Create a directory where the file should go so WriteFile fails
	// (writing to a path that is a directory).
	dir := t.TempDir()
	workflowDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Create deploy-aws.yml as a directory so WriteFile fails.
	filePath := filepath.Join(workflowDir, "deploy-aws.yml")
	if err := os.MkdirAll(filePath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := GenerateWorkflow(config, dir)
	if err == nil {
		t.Fatal("expected error when file write fails")
	}
	if !strings.Contains(err.Error(), "writing workflow file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "writing workflow file")
	}
}

func TestGenerateWorkflow_MissingRequiredConfig(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*WorkflowConfig)
		want   string
	}{
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
			name:   "missing RoleARN",
			modify: func(c *WorkflowConfig) { c.RoleARN = "" },
			want:   "roleARN",
		},
		{
			name:   "missing AccountID",
			modify: func(c *WorkflowConfig) { c.AccountID = "" },
			want:   "accountID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validWorkflowConfig()
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
