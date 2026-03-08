package gcp

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirerun/mint/internal/deploy"
)

// Mock implementations for all interfaces.

type mockRegistry struct {
	repoPath string
	err      error
}

func (m *mockRegistry) EnsureRepository(_ context.Context, _, _, _ string) (string, error) {
	return m.repoPath, m.err
}

type mockBuilder struct {
	imageURI string
	err      error
}

func (m *mockBuilder) BuildImage(_ context.Context, _, imageURI string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.imageURI != "" {
		return m.imageURI, nil
	}
	return imageURI + "@sha256:abc123", nil
}

type mockCloudRun struct {
	info *ServiceInfo
	err  error
}

func (m *mockCloudRun) EnsureService(_ context.Context, _ ServiceOptions) (*ServiceInfo, error) {
	return m.info, m.err
}

type mockIAM struct {
	err error
}

func (m *mockIAM) ConfigureIAMPolicy(_ context.Context, _, _, _ string, _ bool) error {
	return m.err
}

type mockSecrets struct {
	err error
}

func (m *mockSecrets) EnsureSecrets(_ context.Context, _, _, _ string, _ map[string]string) error {
	return m.err
}

type mockSourceRepo struct {
	url string
	err error
}

func (m *mockSourceRepo) EnsureRepo(_ context.Context, _, _ string) (string, error) {
	return m.url, m.err
}

type mockGit struct {
	err error
}

func (m *mockGit) Push(_ context.Context, _, _ string) error {
	return m.err
}

func newTestDeployer(healthServer *httptest.Server) *Deployer {
	client := http.DefaultClient
	if healthServer != nil {
		client = healthServer.Client()
	}
	return &Deployer{
		Registry: &mockRegistry{repoPath: "us-central1-docker.pkg.dev/myproject/myrepo"},
		Builder:  &mockBuilder{},
		CloudRun: &mockCloudRun{info: &ServiceInfo{
			URL:          "https://mysvc-abc123.a.run.app",
			RevisionName: "mysvc-00001-abc",
		}},
		IAM:     &mockIAM{},
		Secrets: &mockSecrets{},
		HealthCheck: &HealthChecker{
			Client:   client,
			Timeout:  1 * time.Second,
			Interval: 100 * time.Millisecond,
		},
		SourceRepo: &mockSourceRepo{url: "https://source.developers.google.com/p/myproject/r/mysvc"},
		Git:        &mockGit{},
		Stderr:     &bytes.Buffer{},
	}
}

func newTestInput() DeployInput {
	return DeployInput{
		Config: &deploy.DeployConfig{
			ProjectID:            "myproject",
			Region:               "us-central1",
			ServiceName:          "mysvc",
			Port:                 8080,
			EnvVars:              map[string]string{"KEY": "value"},
			AllowUnauthenticated: true,
			MinInstances:         0,
			MaxInstances:         10,
			Memory:               "512Mi",
			CPU:                  "1",
			SourceDir:            "/tmp/src",
		},
		MintVersion: "1.0.0",
		SpecHash:    "deadbeef",
		CommitSHA:   "abc1234",
	}
}

func TestDeployFullSuccess(t *testing.T) {
	// Start a healthy HTTP server for health checks.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestDeployer(srv)
	// Override CloudRun mock to return the test server URL.
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}

	input := newTestInput()
	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ServiceURL != srv.URL {
		t.Errorf("ServiceURL = %q, want %q", out.ServiceURL, srv.URL)
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
	d := newTestDeployer(nil)
	d.Registry = &mockRegistry{err: fmt.Errorf("permission denied")}

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
	d := newTestDeployer(nil)
	d.Builder = &mockBuilder{err: fmt.Errorf("build timeout")}

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
	d := newTestDeployer(nil)
	d.CloudRun = &mockCloudRun{err: fmt.Errorf("quota exceeded")}

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
	// Start a server that always returns 503.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	d := newTestDeployer(srv)
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:              srv.URL,
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
	if out.ServiceURL != srv.URL {
		t.Errorf("ServiceURL = %q, want %q", out.ServiceURL, srv.URL)
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestDeployer(srv)
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}
	// Set source repo to error to prove it is not called.
	d.SourceRepo = &mockSourceRepo{err: fmt.Errorf("should not be called")}

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestDeployer(srv)
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}

	input := newTestInput()
	input.Config.Secrets = map[string]string{"API_KEY": "projects/myproject/secrets/api-key"}

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := newTestDeployer(srv)
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}
	// Set secrets client to error to prove it is not called.
	d.Secrets = &mockSecrets{err: fmt.Errorf("should not be called")}

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var capturedImageURI string
	d := newTestDeployer(srv)
	d.Builder = &mockBuilder{imageURI: ""}
	// Use a custom builder to capture the image URI.
	d.Builder = &capturingBuilder{captured: &capturedImageURI}
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var capturedImageURI string
	d := newTestDeployer(srv)
	d.Builder = &capturingBuilder{captured: &capturedImageURI}
	d.CloudRun = &mockCloudRun{info: &ServiceInfo{
		URL:          srv.URL,
		RevisionName: "mysvc-00001-abc",
	}}

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

// capturingBuilder records the image URI passed to BuildImage.
type capturingBuilder struct {
	captured *string
}

func (b *capturingBuilder) BuildImage(_ context.Context, _, imageURI string) (string, error) {
	*b.captured = imageURI
	return imageURI + "@sha256:abc123", nil
}
