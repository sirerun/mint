package azure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		ServiceName:    "my-service",
		Region:         "eastus",
		ResourceGroup:  "my-rg",
		SubscriptionID: "00000000-0000-0000-0000-000000000000",
		RepoOwner:      "sirerun",
		RepoName:       "mint",
		BranchName:     "main",
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
		{"service name", config.ServiceName},
		{"resource group", config.ResourceGroup},
		{"branch name", config.BranchName},
		{"azure/login@v2", "azure/login@v2"},
		{"AZURE_CLIENT_ID", "AZURE_CLIENT_ID"},
		{"AZURE_TENANT_ID", "AZURE_TENANT_ID"},
		{"AZURE_SUBSCRIPTION_ID", "AZURE_SUBSCRIPTION_ID"},
		{"Container Apps", "containerapp update"},
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

	expected := filepath.Join(dir, ".github", "workflows", "deploy-azure.yml")
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

	dir := t.TempDir()
	workflowDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Create deploy-azure.yml as a directory so WriteFile fails.
	filePath := filepath.Join(workflowDir, "deploy-azure.yml")
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
			name:   "missing ServiceName",
			modify: func(c *WorkflowConfig) { c.ServiceName = "" },
			want:   "serviceName",
		},
		{
			name:   "missing Region",
			modify: func(c *WorkflowConfig) { c.Region = "" },
			want:   "region",
		},
		{
			name:   "missing ResourceGroup",
			modify: func(c *WorkflowConfig) { c.ResourceGroup = "" },
			want:   "resourceGroup",
		},
		{
			name:   "missing SubscriptionID",
			modify: func(c *WorkflowConfig) { c.SubscriptionID = "" },
			want:   "subscriptionID",
		},
		{
			name:   "missing RepoOwner",
			modify: func(c *WorkflowConfig) { c.RepoOwner = "" },
			want:   "repoOwner",
		},
		{
			name:   "missing RepoName",
			modify: func(c *WorkflowConfig) { c.RepoName = "" },
			want:   "repoName",
		},
		{
			name:   "missing BranchName",
			modify: func(c *WorkflowConfig) { c.BranchName = "" },
			want:   "branchName",
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

func TestGenerateWorkflow_OIDCFederation(t *testing.T) {
	dir := t.TempDir()
	config := validWorkflowConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	// Verify the workflow uses OIDC federation via azure/login with federated identity fields.
	checks := []string{
		"id-token: write",
		"azure/login@v2",
		"client-id:",
		"tenant-id:",
		"subscription-id:",
	}
	for _, want := range checks {
		if !strings.Contains(result.Content, want) {
			t.Errorf("workflow missing OIDC element %q", want)
		}
	}
}

func TestGenerateWorkflow_ACRBuildAndPush(t *testing.T) {
	dir := t.TempDir()
	config := validWorkflowConfig()

	result, err := GenerateWorkflow(config, dir)
	if err != nil {
		t.Fatalf("GenerateWorkflow() error = %v", err)
	}

	checks := []string{
		"acr login",
		"docker build",
		"docker push",
		config.ServiceName + "acr.azurecr.io",
	}
	for _, want := range checks {
		if !strings.Contains(result.Content, want) {
			t.Errorf("workflow missing ACR element %q", want)
		}
	}
}
