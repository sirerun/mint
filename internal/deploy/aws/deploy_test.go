package aws

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sirerun/mint/internal/deploy"
)

// Mock implementations for orchestrator interfaces.

type mockRegistryProvisioner struct {
	repoURI string
	err     error
}

func (m *mockRegistryProvisioner) EnsureRepository(_ context.Context, _, _ string) (string, error) {
	return m.repoURI, m.err
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

func (m *mockIAMConfigurator) ConfigureIAM(_ context.Context, _, _ string, _ bool) error {
	return m.err
}

type mockSecretProvisioner struct {
	arns []string
	err  error
}

func (m *mockSecretProvisioner) EnsureSecrets(_ context.Context, _, _ string, _ map[string]string) ([]string, error) {
	return m.arns, m.err
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
		Registry: &mockRegistryProvisioner{repoURI: "123456789.dkr.ecr.us-east-1.amazonaws.com/mysvc"},
		Builder:  &mockImageBuilder{},
		ECS: &mockServiceDeployer{info: &DeployServiceInfo{
			URL:     "https://mysvc.us-east-1.elb.amazonaws.com",
			TaskARN: "arn:aws:ecs:us-east-1:123456789:task/mysvc/abc123",
		}},
		IAM:     &mockIAMConfigurator{},
		Secrets: &mockSecretProvisioner{arns: []string{"arn:aws:secretsmanager:us-east-1:123456789:secret:test"}},
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
			Region:       "us-east-1",
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
			ProjectID:    "unused",
		},
		MintVersion: "1.0.0",
		SpecHash:    "deadbeef",
		CommitSHA:   "abc1234",
	}
}

func TestDeployFullSuccess(t *testing.T) {
	d := newTestDeployer()
	var stderr bytes.Buffer
	d.Stderr = &stderr

	input := newTestInput()

	out, err := d.Deploy(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ServiceURL != "https://mysvc.us-east-1.elb.amazonaws.com" {
		t.Errorf("ServiceURL = %q, want %q", out.ServiceURL, "https://mysvc.us-east-1.elb.amazonaws.com")
	}
	if out.TaskARN != "arn:aws:ecs:us-east-1:123456789:task/mysvc/abc123" {
		t.Errorf("TaskARN = %q, want %q", out.TaskARN, "arn:aws:ecs:us-east-1:123456789:task/mysvc/abc123")
	}
	if out.ImageURI == "" {
		t.Error("ImageURI should not be empty")
	}
	if !out.Healthy {
		t.Error("Healthy should be true")
	}

	logs := stderr.String()
	expectedLogs := []string{
		"Provisioning ECR repository...",
		"Building container image...",
		"Configuring IAM roles...",
		"Deploying to ECS Fargate...",
		"Running health check...",
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

	_, err := d.Deploy(context.Background(), newTestInput())
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

	_, err := d.Deploy(context.Background(), newTestInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "build:") {
		t.Errorf("error = %q, want it to contain 'build:'", err.Error())
	}
}

func TestDeployECSFailure(t *testing.T) {
	d := newTestDeployer()
	d.ECS = &mockServiceDeployer{err: fmt.Errorf("quota exceeded")}

	_, err := d.Deploy(context.Background(), newTestInput())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ecs:") {
		t.Errorf("error = %q, want it to contain 'ecs:'", err.Error())
	}
}

func TestDeployIAMFailure(t *testing.T) {
	d := newTestDeployer()
	d.IAM = &mockIAMConfigurator{err: fmt.Errorf("iam policy update failed")}

	_, err := d.Deploy(context.Background(), newTestInput())
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

func TestDeployHealthCheckUnhealthy(t *testing.T) {
	d := newTestDeployer()
	d.Health = &mockHealthProber{result: &HealthProbeResult{
		Healthy:    false,
		StatusCode: 503,
		Message:    "health check timed out",
	}}
	d.ECS = &mockServiceDeployer{info: &DeployServiceInfo{
		URL:             "https://mysvc.us-east-1.elb.amazonaws.com",
		TaskARN:         "arn:aws:ecs:us-east-1:123456789:task/mysvc/def456",
		PreviousTaskARN: "arn:aws:ecs:us-east-1:123456789:task/mysvc/abc123",
	}}

	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), newTestInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Healthy {
		t.Error("Healthy should be false when health check fails")
	}

	logs := stderr.String()
	if !strings.Contains(logs, "Warning: service unhealthy") {
		t.Errorf("stderr should contain unhealthy warning, got: %s", logs)
	}
	if !strings.Contains(logs, "abc123") {
		t.Errorf("stderr should mention previous task, got: %s", logs)
	}
}

func TestDeployHealthCheckError(t *testing.T) {
	d := newTestDeployer()
	d.Health = &mockHealthProber{err: fmt.Errorf("connection refused")}

	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), newTestInput())
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

func TestDeploySecretsSkippedWhenNone(t *testing.T) {
	d := newTestDeployer()
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

// capturingServiceDeployer records the options passed to EnsureService.
type capturingServiceDeployer struct {
	captured DeployServiceOptions
}

func (m *capturingServiceDeployer) EnsureService(_ context.Context, opts DeployServiceOptions) (*DeployServiceInfo, error) {
	m.captured = opts
	return &DeployServiceInfo{
		URL:     "https://test.elb.amazonaws.com",
		TaskARN: "arn:aws:ecs:us-east-1:123456789:task/test/001",
	}, nil
}

func TestDeployConfigForwardedToECS(t *testing.T) {
	captor := &capturingServiceDeployer{}
	d := newTestDeployer()
	d.ECS = captor

	input := newTestInput()
	input.Config.Region = "eu-west-1"
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

	if captor.captured.Region != "eu-west-1" {
		t.Errorf("Region = %q, want %q", captor.captured.Region, "eu-west-1")
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
	if captor.captured.AllowPublic != false {
		t.Error("AllowPublic should be false")
	}
	if captor.captured.EnvVars["FOO"] != "bar" {
		t.Errorf("EnvVars[FOO] = %q, want %q", captor.captured.EnvVars["FOO"], "bar")
	}
}

func TestDeployNilStderr(t *testing.T) {
	d := newTestDeployer()
	d.Stderr = nil

	out, err := d.Deploy(context.Background(), newTestInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ServiceURL == "" {
		t.Error("ServiceURL should not be empty")
	}
}

func TestDeployIdempotency(t *testing.T) {
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
	if out1.TaskARN != out2.TaskARN {
		t.Errorf("TaskARN mismatch: first=%q second=%q", out1.TaskARN, out2.TaskARN)
	}
	if out1.ImageURI != out2.ImageURI {
		t.Errorf("ImageURI mismatch: first=%q second=%q", out1.ImageURI, out2.ImageURI)
	}
	if out1.Healthy != out2.Healthy {
		t.Errorf("Healthy mismatch: first=%v second=%v", out1.Healthy, out2.Healthy)
	}
}

func TestDeployHealthCheckUnhealthyNoPreviousTask(t *testing.T) {
	d := newTestDeployer()
	d.Health = &mockHealthProber{result: &HealthProbeResult{
		Healthy:    false,
		StatusCode: 500,
		Message:    "internal server error",
	}}
	d.ECS = &mockServiceDeployer{info: &DeployServiceInfo{
		URL:             "https://mysvc.us-east-1.elb.amazonaws.com",
		TaskARN:         "arn:aws:ecs:us-east-1:123456789:task/mysvc/abc123",
		PreviousTaskARN: "",
	}}

	var stderr bytes.Buffer
	d.Stderr = &stderr

	out, err := d.Deploy(context.Background(), newTestInput())
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
	if strings.Contains(logs, "previous task") {
		t.Errorf("stderr should not mention previous task when there is none, got: %s", logs)
	}
}

// capturingSecretProvisioner records the secrets map passed to EnsureSecrets.
type capturingSecretProvisioner struct {
	captured map[string]string
}

func (m *capturingSecretProvisioner) EnsureSecrets(_ context.Context, _, _ string, secrets map[string]string) ([]string, error) {
	m.captured = secrets
	return nil, nil
}

func TestDeploySecretsConversion(t *testing.T) {
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
