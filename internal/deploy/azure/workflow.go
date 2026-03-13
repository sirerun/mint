package azure

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// WorkflowConfig holds configuration for generating a GitHub Actions workflow
// that deploys to Azure Container Apps.
type WorkflowConfig struct {
	ServiceName    string
	Region         string
	ResourceGroup  string
	SubscriptionID string
	RepoOwner      string
	RepoName       string
	BranchName     string
}

// WorkflowResult holds the output of workflow generation.
type WorkflowResult struct {
	FilePath string // path where workflow was written
	Content  string // the YAML content
}

// workflowTemplateStr is a variable to allow tests to inject invalid templates.
var workflowTemplateStr = workflowTemplateConst

const workflowTemplateConst = `name: Deploy to Azure Container Apps
on:
  push:
    branches: [{{.BranchName}}]
jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    env:
      AZURE_CLIENT_ID: ${{"{{"}} secrets.AZURE_CLIENT_ID {{"}}"}}
      AZURE_TENANT_ID: ${{"{{"}} secrets.AZURE_TENANT_ID {{"}}"}}
      AZURE_SUBSCRIPTION_ID: ${{"{{"}} secrets.AZURE_SUBSCRIPTION_ID {{"}}"}}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: azure/login@v2
        with:
          client-id: ${{"{{"}} secrets.AZURE_CLIENT_ID {{"}}"}}
          tenant-id: ${{"{{"}} secrets.AZURE_TENANT_ID {{"}}"}}
          subscription-id: ${{"{{"}} secrets.AZURE_SUBSCRIPTION_ID {{"}}"}}
      - name: Install mint
        run: go install github.com/sirerun/mint@latest
      - name: Build and push to ACR
        run: |
          az acr login --name {{.ServiceName}}acr
          docker build -t {{.ServiceName}}acr.azurecr.io/{{.ServiceName}}:${{"{{"}} github.sha {{"}}"}} .
          docker push {{.ServiceName}}acr.azurecr.io/{{.ServiceName}}:${{"{{"}} github.sha {{"}}"}}
      - name: Deploy to Container Apps
        run: |
          az containerapp update \
            --name {{.ServiceName}} \
            --resource-group {{.ResourceGroup}} \
            --image {{.ServiceName}}acr.azurecr.io/{{.ServiceName}}:${{"{{"}} github.sha {{"}}"}}
`

// GenerateWorkflow generates a GitHub Actions workflow YAML file for deploying
// to Azure Container Apps. It writes the file to <outputDir>/.github/workflows/deploy-azure.yml,
// creating directories as needed.
func GenerateWorkflow(config WorkflowConfig, outputDir string) (*WorkflowResult, error) {
	if err := validateWorkflowConfig(config); err != nil {
		return nil, err
	}

	tmpl, err := template.New("workflow").Parse(workflowTemplateStr)
	if err != nil {
		return nil, fmt.Errorf("parsing workflow template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, config); err != nil {
		return nil, fmt.Errorf("executing workflow template: %w", err)
	}

	dir := filepath.Join(outputDir, ".github", "workflows")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating workflow directory: %w", err)
	}

	filePath := filepath.Join(dir, "deploy-azure.yml")
	content := buf.String()
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("writing workflow file: %w", err)
	}

	return &WorkflowResult{
		FilePath: filePath,
		Content:  content,
	}, nil
}

func validateWorkflowConfig(config WorkflowConfig) error {
	switch {
	case config.ServiceName == "":
		return fmt.Errorf("serviceName is required")
	case config.Region == "":
		return fmt.Errorf("region is required")
	case config.ResourceGroup == "":
		return fmt.Errorf("resourceGroup is required")
	case config.SubscriptionID == "":
		return fmt.Errorf("subscriptionID is required")
	case config.RepoOwner == "":
		return fmt.Errorf("repoOwner is required")
	case config.RepoName == "":
		return fmt.Errorf("repoName is required")
	case config.BranchName == "":
		return fmt.Errorf("branchName is required")
	}
	return nil
}
