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

func TestDeployIAMFailure(t *testing.T) {
	d := newTestDeployer()
	d.IAM = &mockIAMConfigurator{err: fmt.Errorf("iam policy update failed")}

	input := newTestInput()

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "iam:") {
		t.Errorf("error = %q, want it to contain 'iam:'", err.Error())
	}
	if !strings.Contains(err.Error(), "iam policy update failed") {
		t.Errorf("error = %q, want it to contain 'iam policy update failed'", err.Error())
	}
}

func TestDeploySecretsFailure(t *testing.T) {
	d := newTestDeployer()
	d.Secrets = &mockSecretProvisioner{err: fmt.Errorf("secret access denied")}

	input := newTestInput()
	input.Config.Secrets = []deploy.SecretMapping{
		{EnvVar: "DB_PASS", SecretName: "db-password"},
	}

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "secrets:") {
		t.Errorf("error = %q, want it to contain 'secrets:'", err.Error())
	}
	if !strings.Contains(err.Error(), "secret access denied") {
		t.Errorf("error = %q, want it to contain 'secret access denied'", err.Error())
	}
}

func TestDeploySourceRepoFailure(t *testing.T) {
	d := newTestDeployer()
	d.SourceRepo = &mockRepoProvisioner{err: fmt.Errorf("repo creation failed")}

	input := newTestInput()
	input.Config.NoSourceRepo = false

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "source repo:") {
		t.Errorf("error = %q, want it to contain 'source repo:'", err.Error())
	}
}

func TestDeployGitPushFailure(t *testing.T) {
	d := newTestDeployer()
	d.Git = &mockSourcePusher{err: fmt.Errorf("git push rejected")}

	input := newTestInput()
	input.Config.NoSourceRepo = false

	_, err := d.Deploy(context.Background(), input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "git push:") {
		t.Errorf("error = %q, want it to contain 'git push:'", err.Error())
	}
}

func TestDeployHealthCheckError(t *testing.T) {
	d := newTestDeployer()
	d.Health = &mockHealthProber{err: fmt.Errorf("connection refused")}

	input := newTestInput()
	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("health check error should not fail deploy, got: %v", err)
	}
	if out.Healthy {
		t.Error("Healthy should be false when health check returns error")
	}
	logs := stderr.String()
	if !strings.Contains(logs, "Warning: health check error") {
		t.Errorf("stderr should contain health check error warning, got: %s", logs)
	}
	if !strings.Contains(logs, "connection refused") {
		t.Errorf("stderr should contain error message, got: %s", logs)
	}
}

func TestDeployHealthCheckUnhealthyNoPreviousRevision(t *testing.T) {
	d := newTestDeployer()
	d.Health = &mockHealthProber{result: &HealthProbeResult{
		Healthy:    false,
		StatusCode: 500,
		Message:    "internal server error",
	}}
	d.CloudRun = &mockServiceDeployer{info: &DeployServiceInfo{
		URL:              "https://mysvc.a.run.app",
		RevisionName:     "mysvc-00001-abc",
		PreviousRevision: "", // no previous revision
	}}

	input := newTestInput()
	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Healthy {
		t.Error("Healthy should be false")
	}
	logs := stderr.String()
	if !strings.Contains(logs, "Warning: service unhealthy") {
		t.Errorf("stderr should contain unhealthy warning, got: %s", logs)
	}
	// Should NOT mention previous revision when there is none.
	if strings.Contains(logs, "previous revision") {
		t.Errorf("stderr should not mention previous revision when there is none, got: %s", logs)
	}
}

func TestDeployIdempotency(t *testing.T) {
	// Deploy twice with same input, verify both succeed without error.
	d := newTestDeployer()
	input := newTestInput()

	out1, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("first deploy: unexpected error: %v", err)
	}

	out2, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("second deploy: unexpected error: %v", err)
	}

	if out1.ServiceURL != out2.ServiceURL {
		t.Errorf("ServiceURL mismatch: first=%q second=%q", out1.ServiceURL, out2.ServiceURL)
	}
	if out1.RevisionName != out2.RevisionName {
		t.Errorf("RevisionName mismatch: first=%q second=%q", out1.RevisionName, out2.RevisionName)
	}
	if out1.ImageURI != out2.ImageURI {
		t.Errorf("ImageURI mismatch: first=%q second=%q", out1.ImageURI, out2.ImageURI)
	}
	if out1.Healthy != out2.Healthy {
		t.Errorf("Healthy mismatch: first=%v second=%v", out1.Healthy, out2.Healthy)
	}
}

// capturingSecretProvisioner records the secrets map passed to EnsureSecrets.
type capturingSecretProvisioner struct {
	captured map[string]string
}

func (m *capturingSecretProvisioner) EnsureSecrets(_ context.Context, _, _, _ string, secrets map[string]string) error {
	m.captured = secrets
	return nil
}

func TestDeploySecretsConversion(t *testing.T) {
	// Verify that []SecretMapping is correctly converted to map[string]string.
	captor := &capturingSecretProvisioner{}
	d := newTestDeployer()
	d.Secrets = captor

	input := newTestInput()
	input.Config.Secrets = []deploy.SecretMapping{
		{EnvVar: "DB_PASSWORD", SecretName: "db-pass-secret"},
		{EnvVar: "API_KEY", SecretName: "api-key-secret"},
		{EnvVar: "TOKEN", SecretName: "auth-token"},
	}

	_, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(captor.captured) != 3 {
		t.Fatalf("expected 3 secrets, got %d", len(captor.captured))
	}

	expected := map[string]string{
		"DB_PASSWORD": "db-pass-secret",
		"API_KEY":     "api-key-secret",
		"TOKEN":       "auth-token",
	}
	for k, v := range expected {
		if captor.captured[k] != v {
			t.Errorf("secret %q = %q, want %q", k, captor.captured[k], v)
		}
	}
}

func TestDeployNilStderr(t *testing.T) {
	// Verify that deploy works when Stderr is nil (no log output).
	d := newTestDeployer()
	d.Stderr = nil

	input := newTestInput()

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ServiceURL == "" {
		t.Error("ServiceURL should not be empty")
	}
}

func TestDeployNoSecretsNoSourceRepo(t *testing.T) {
	// Verify that deploy works when both secrets and source repo are disabled.
	d := newTestDeployer()
	// Set both to error to prove they are not called.
	d.Secrets = &mockSecretProvisioner{err: fmt.Errorf("should not be called")}
	d.SourceRepo = &mockRepoProvisioner{err: fmt.Errorf("should not be called")}
	d.Git = &mockSourcePusher{err: fmt.Errorf("should not be called")}

	input := newTestInput()
	input.Config.Secrets = nil
	input.Config.NoSourceRepo = true

	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RepoURL != "" {
		t.Errorf("RepoURL = %q, want empty", out.RepoURL)
	}

	logs := stderr.String()
	if strings.Contains(logs, "Configuring secrets") {
		t.Error("should not log secrets")
	}
	if strings.Contains(logs, "Pushing source") {
		t.Error("should not log source push")
	}
}

// capturingServiceDeployer records the options passed to EnsureService.
type capturingServiceDeployer struct {
	captured DeployServiceOptions
}

func (m *capturingServiceDeployer) EnsureService(_ context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error) {
	m.captured = opts
	return &DeployServiceInfo{
		URL:          "https://test.run.app",
		RevisionName: "test-00001",
	}, nil
}

func TestDeployPassesConfigToCloudRun(t *testing.T) {
	// Verify that deploy options are correctly forwarded from config to CloudRun.
	captor := &capturingServiceDeployer{}
	d := newTestDeployer()
	d.CloudRun = captor

	input := newTestInput()
	input.Config.ProjectID = "test-project"
	input.Config.Region = "europe-west1"
	input.Config.ServiceName = "test-svc"
	input.Config.Port = 9090
	input.Config.EnvVars = map[string]string{"FOO": "bar"}
	input.Config.MinInstances = 1
	input.Config.MaxInstances = 5
	input.Config.Memory = "1Gi"
	input.Config.CPU = "2"
	input.Config.Public = false

	_, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captor.captured.ProjectID != "test-project" {
		t.Errorf("ProjectID = %q, want %q", captor.captured.ProjectID, "test-project")
	}
	if captor.captured.Region != "europe-west1" {
		t.Errorf("Region = %q, want %q", captor.captured.Region, "europe-west1")
	}
	if captor.captured.ServiceName != "test-svc" {
		t.Errorf("ServiceName = %q, want %q", captor.captured.ServiceName, "test-svc")
	}
	if captor.captured.Port != 9090 {
		t.Errorf("Port = %d, want %d", captor.captured.Port, 9090)
	}
	if captor.captured.MinInstances != 1 {
		t.Errorf("MinInstances = %d, want %d", captor.captured.MinInstances, 1)
	}
	if captor.captured.MaxInstances != 5 {
		t.Errorf("MaxInstances = %d, want %d", captor.captured.MaxInstances, 5)
	}
	if captor.captured.Memory != "1Gi" {
		t.Errorf("Memory = %q, want %q", captor.captured.Memory, "1Gi")
	}
	if captor.captured.CPU != "2" {
		t.Errorf("CPU = %q, want %q", captor.captured.CPU, "2")
	}
	if captor.captured.AllowUnauthenticated != false {
		t.Error("AllowUnauthenticated should be false")
	}
	if captor.captured.EnvVars["FOO"] != "bar" {
		t.Errorf("EnvVars[FOO] = %q, want %q", captor.captured.EnvVars["FOO"], "bar")
	}
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
