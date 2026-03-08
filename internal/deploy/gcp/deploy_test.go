package gcp

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirerun/mint/internal/deploy"
)

// Mock implementations for orchestrator interfaces.

type mockRegistryProvisioner struct {
	repoPath string
	err      error
}

func (m *mockRegistryProvisioner) EnsureRepository(_ context.Context, _, _, _ string) (string, error) {
	return m.repoPath, m.err
}

type mockImageBuilder struct {
	imageURI string
	err      error
}

func (m *mockImageBuilder) BuildImage(_ context.Context, _, imageURI string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.imageURI != "" {
		return m.imageURI, nil
	}
	return imageURI + "@sha256:abc123", nil
}

type mockServiceDeployer struct {
	info *DeployServiceInfo
	err  error
}

func (m *mockServiceDeployer) EnsureService(_ context.Context, _ DeployServiceOptions) (*DeployServiceInfo, error) {
	return m.info, m.err
}

type mockIAMConfigurator struct {
	err error
}

func (m *mockIAMConfigurator) ConfigureIAMPolicy(_ context.Context, _, _, _ string, _ bool) error {
	return m.err
}

type mockSecretProvisioner struct {
	err error
}

func (m *mockSecretProvisioner) EnsureSecrets(_ context.Context, _, _, _ string, _ map[string]string) error {
	return m.err
}

type mockRepoProvisioner struct {
	url string
	err error
}

func (m *mockRepoProvisioner) EnsureRepo(_ context.Context, _, _ string) (string, error) {
	return m.url, m.err
}

type mockSourcePusher struct {
	err error
}

func (m *mockSourcePusher) Push(_ context.Context, _, _ string) error {
	return m.err
}

type mockHealthProber struct {
	result *HealthProbeResult
	err    error
}

func (m *mockHealthProber) Check(_ context.Context, _ string) (*HealthProbeResult, error) {
	return m.result, m.err
}

func newTestDeployer() *Deployer {
	return &Deployer{
		Registry: &mockRegistryProvisioner{repoPath: "us-central1-docker.pkg.dev/myproject/myrepo"},
		Builder:  &mockImageBuilder{},
		CloudRun: &mockServiceDeployer{info: &DeployServiceInfo{
			URL:          "https://mysvc-abc123.a.run.app",
			RevisionName: "mysvc-00001-abc",
		}},
		IAM:        &mockIAMConfigurator{},
		Secrets:    &mockSecretProvisioner{},
		SourceRepo: &mockRepoProvisioner{url: "https://source.developers.google.com/p/myproject/r/mysvc"},
		Git:        &mockSourcePusher{},
		Health: &mockHealthProber{result: &HealthProbeResult{
			Healthy:    true,
			StatusCode: 200,
			Message:    "service is healthy",
		}},
		Stderr: &bytes.Buffer{},
	}
}

func newTestInput() DeployInput {
	return DeployInput{
		Config: &deploy.DeployConfig{
			ProjectID:    "myproject",
			Region:       "us-central1",
			ServiceName:  "mysvc",
			Port:         8080,
			EnvVars:      map[string]string{"KEY": "value"},
			Public:       true,
			MinInstances: 0,
			MaxInstances: 10,
			Memory:       "512Mi",
			CPU:          "1",
			SourceDir:    "/tmp/src",
			Timeout:      300,
		},
		MintVersion: "1.0.0",
		SpecHash:    "deadbeef",
		CommitSHA:   "abc1234",
	}
}

func TestDeployFullSuccess(t *testing.T) {
	d := newTestDeployer()

	input := newTestInput()
	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ServiceURL != "https://mysvc-abc123.a.run.app" {
		t.Errorf("ServiceURL = %q, want %q", out.ServiceURL, "https://mysvc-abc123.a.run.app")
	}
	if out.RevisionName != "mysvc-00001-abc" {
		t.Errorf("RevisionName = %q, want %q", out.RevisionName, "mysvc-00001-abc")
	}
	if out.ImageURI == "" {
		t.Error("ImageURI should not be empty")
	}
	if !out.Healthy {
		t.Error("Healthy should be true")
	}
	if out.RepoURL == "" {
		t.Error("RepoURL should not be empty when NoSourceRepo is false")
	}

	// Verify progress output.
	logs := stderr.String()
	expectedLogs := []string{
		"Provisioning Artifact Registry repository...",
		"Building container image...",
		"Deploying to Cloud Run...",
		"Configuring IAM policy...",
		"Running health check...",
		"Pushing source to Cloud Source Repository...",
		"Deployment complete:",
	}
	for _, expected := range expectedLogs {
		if !strings.Contains(logs, expected) {
			t.Errorf("stderr missing %q", expected)
		}
	}
}

func TestDeployRegistryFailure(t *testing.T) {
	d := newTestDeployer()
	d.Registry = &mockRegistryProvisioner{err: fmt.Errorf("permission denied")}

	input := newTestInput()

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "registry:") {
		t.Errorf("error = %q, want it to contain 'registry:'", err.Error())
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error = %q, want it to contain 'permission denied'", err.Error())
	}
}

func TestDeployBuildFailure(t *testing.T) {
	d := newTestDeployer()
	d.Builder = &mockImageBuilder{err: fmt.Errorf("build timeout")}

	input := newTestInput()

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "build:") {
		t.Errorf("error = %q, want it to contain 'build:'", err.Error())
	}
}

func TestDeployCloudRunFailure(t *testing.T) {
	d := newTestDeployer()
	d.CloudRun = &mockServiceDeployer{err: fmt.Errorf("quota exceeded")}

	input := newTestInput()

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cloud run:") {
		t.Errorf("error = %q, want it to contain 'cloud run:'", err.Error())
	}
}

func TestDeployHealthCheckUnhealthy(t *testing.T) {
	d := newTestDeployer()
	d.Health = &mockHealthProber{result: &HealthProbeResult{
		Healthy:    false,
		StatusCode: 503,
		Message:    "health check timed out",
	}}
	d.CloudRun = &mockServiceDeployer{info: &DeployServiceInfo{
		URL:              "https://mysvc.a.run.app",
		RevisionName:     "mysvc-00002-def",
		PreviousRevision: "mysvc-00001-abc",
	}}

	input := newTestInput()
	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Healthy {
		t.Error("Healthy should be false when health check fails")
	}

	// Should contain warning about unhealthy + previous revision.
	logs := stderr.String()
	if !strings.Contains(logs, "Warning: service unhealthy") {
		t.Errorf("stderr should contain unhealthy warning, got: %s", logs)
	}
	if !strings.Contains(logs, "mysvc-00001-abc") {
		t.Errorf("stderr should mention previous revision, got: %s", logs)
	}
}

func TestDeploySourceRepoSkippedWhenNoSourceRepo(t *testing.T) {
	d := newTestDeployer()
	// Set source repo to error to prove it is not called.
	d.SourceRepo = &mockRepoProvisioner{err: fmt.Errorf("should not be called")}

	input := newTestInput()
	input.Config.NoSourceRepo = true

	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.RepoURL != "" {
		t.Errorf("RepoURL = %q, want empty when NoSourceRepo=true", out.RepoURL)
	}

	logs := stderr.String()
	if strings.Contains(logs, "Pushing source to Cloud Source Repository") {
		t.Error("should not log source repo push when NoSourceRepo=true")
	}
}

func TestDeploySecretsConfigured(t *testing.T) {
	d := newTestDeployer()

	input := newTestInput()
	input.Config.Secrets = []deploy.SecretMapping{
		{EnvVar: "API_KEY", SecretName: "api-key"},
	}

	var stderr bytes.Buffer
	d.Stderr = &stderr

	_, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := stderr.String()
	if !strings.Contains(logs, "Configuring secrets...") {
		t.Error("stderr should contain secrets configuration message")
	}
}

func TestDeploySecretsSkippedWhenNone(t *testing.T) {
	d := newTestDeployer()
	// Set secrets client to error to prove it is not called.
	d.Secrets = &mockSecretProvisioner{err: fmt.Errorf("should not be called")}

	input := newTestInput()
	input.Config.Secrets = nil

	var stderr bytes.Buffer
	d.Stderr = &stderr

	_, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logs := stderr.String()
	if strings.Contains(logs, "Configuring secrets") {
		t.Error("should not log secrets configuration when no secrets configured")
	}
}

func TestDeployUsesCommitSHAForTag(t *testing.T) {
	var capturedImageURI string
	d := newTestDeployer()
	d.Builder = &capturingImageBuilder{captured: &capturedImageURI}

	input := newTestInput()
	input.CommitSHA = "sha123"
	input.SpecHash = "hash456"

	_, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedImageURI, ":sha123") {
		t.Errorf("image URI = %q, want it to use commit SHA as tag", capturedImageURI)
	}
}

func TestDeployUsesSpecHashWhenNoCommitSHA(t *testing.T) {
	var capturedImageURI string
	d := newTestDeployer()
	d.Builder = &capturingImageBuilder{captured: &capturedImageURI}

	input := newTestInput()
	input.CommitSHA = ""
	input.SpecHash = "hash456"

	_, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedImageURI, ":hash456") {
		t.Errorf("image URI = %q, want it to use spec hash as tag", capturedImageURI)
	}
}

// capturingImageBuilder records the image URI passed to BuildImage.
type capturingImageBuilder struct {
	captured *string
}

func (b *capturingImageBuilder) BuildImage(_ context.Context, _, imageURI string) (string, error) {
	*b.captured = imageURI
	return imageURI + "@sha256:abc123", nil
}

func TestDeployHealthCheckWithHTTPServer(t *testing.T) {
	// Integration-style test: use a real HTTP server for health checks.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestDeployer()
	d.CloudRun = &mockServiceDeployer{info: &DeployServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}
	d.Health = &mockHealthProber{result: &HealthProbeResult{
		Healthy:    true,
		StatusCode: 200,
		Message:    "service is healthy",
	}}

	input := newTestInput()

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Healthy {
		t.Error("expected healthy")
	}
}
