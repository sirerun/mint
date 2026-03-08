package gcp

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// WorkflowConfig holds configuration for generating a GitHub Actions workflow.
type WorkflowConfig struct {
	ProjectID                string
	Region                   string
	ServiceName              string
	SourceDir                string
	WorkloadIdentityProvider string // full resource name
	ServiceAccountEmail      string
	SpecPath                 string // path to OpenAPI spec (optional, for regeneration)
}

// WorkflowResult holds the output.
type WorkflowResult struct {
	FilePath string // path where workflow was written
	Content  string // the YAML content
}

const workflowTemplate = `name: Deploy to Cloud Run
on:
  push:
    branches: [main]
    paths:
      - '{{.SourceDir}}/**'
jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: '{{.WorkloadIdentityProvider}}'
          service_account: '{{.ServiceAccountEmail}}'
      - name: Install mint
        run: go install github.com/sirerun/mint@latest
      - name: Deploy
        run: mint deploy gcp --project {{.ProjectID}} --region {{.Region}} --source {{.SourceDir}} --service {{.ServiceName}}
`

// GenerateWorkflow generates a GitHub Actions workflow YAML file for deploying
// to Google Cloud Run. It writes the file to <outputDir>/.github/workflows/deploy-gcp.yml,
// creating directories as needed.
func GenerateWorkflow(config WorkflowConfig, outputDir string) (*WorkflowResult, error) {
	if err := validateWorkflowConfig(config); err != nil {
		return nil, err
	}

	tmpl, err := template.New("workflow").Parse(workflowTemplate)
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

	filePath := filepath.Join(dir, "deploy-gcp.yml")
	content := buf.String()
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("writing workflow file: %w", err)
	}

	return &WorkflowResult{
		FilePath: filePath,
		Content:  content,
	}, nil
}

func validateWorkflowConfig(c WorkflowConfig) error {
	switch {
	case c.ProjectID == "":
		return fmt.Errorf("projectID is required")
	case c.Region == "":
		return fmt.Errorf("region is required")
	case c.ServiceName == "":
		return fmt.Errorf("serviceName is required")
	case c.SourceDir == "":
		return fmt.Errorf("sourceDir is required")
	case c.WorkloadIdentityProvider == "":
		return fmt.Errorf("workloadIdentityProvider is required")
	case c.ServiceAccountEmail == "":
		return fmt.Errorf("serviceAccountEmail is required")
	}
	return nil
}
