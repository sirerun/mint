package gcp

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// mockSecretClient implements SecretClient for testing.
type mockSecretClient struct {
	existing  map[string]bool  // secrets that already exist
	getErr    map[string]error // forced errors from GetSecret
	createErr map[string]error // forced errors from CreateSecret
	created   []string         // secrets that were created
}

func newMockClient() *mockSecretClient {
	return &mockSecretClient{
		existing:  make(map[string]bool),
		getErr:    make(map[string]error),
		createErr: make(map[string]error),
	}
}

func (m *mockSecretClient) GetSecret(_ context.Context, name string) error {
	if err, ok := m.getErr[name]; ok {
		return err
	}
	if m.existing[name] {
		return nil
	}
	return &NotFoundErr{Name: name}
}

func (m *mockSecretClient) CreateSecret(_ context.Context, _, secretID string) error {
	if err, ok := m.createErr[secretID]; ok {
		return err
	}
	m.created = append(m.created, secretID)
	return nil
}

func TestSecretResourceName(t *testing.T) {
	tests := []struct {
		projectID string
		secretID  string
		want      string
	}{
		{"my-project", "api-key", "projects/my-project/secrets/api-key"},
		{"proj-123", "db-password", "projects/proj-123/secrets/db-password"},
	}
	for _, tt := range tests {
		got := SecretResourceName(tt.projectID, tt.secretID)
		if got != tt.want {
			t.Errorf("SecretResourceName(%q, %q) = %q, want %q", tt.projectID, tt.secretID, got, tt.want)
		}
	}
}

func TestEnsureSecrets_AllExist(t *testing.T) {
	client := newMockClient()
	client.existing["projects/proj/secrets/key-a"] = true
	client.existing["projects/proj/secrets/key-b"] = true

	config := SecretConfig{
		ProjectID: "proj",
		Secrets: []SecretMount{
			{EnvVar: "KEY_A", SecretName: "key-a"},
			{EnvVar: "KEY_B", SecretName: "key-b"},
		},
	}

	var stderr bytes.Buffer
	mounts, err := EnsureSecrets(context.Background(), client, config, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mounts) != 2 {
		t.Errorf("got %d mounts, want 2", len(mounts))
	}
	if len(client.created) != 0 {
		t.Errorf("expected no secrets created, got %v", client.created)
	}
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output, got %q", stderr.String())
	}
}

func TestEnsureSecrets_CreatesNotFound(t *testing.T) {
	client := newMockClient()
	// key-a does not exist

	config := SecretConfig{
		ProjectID: "proj",
		Secrets: []SecretMount{
			{EnvVar: "KEY_A", SecretName: "key-a"},
		},
	}

	var stderr bytes.Buffer
	mounts, err := EnsureSecrets(context.Background(), client, config, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mounts) != 1 {
		t.Errorf("got %d mounts, want 1", len(mounts))
	}
	if len(client.created) != 1 || client.created[0] != "key-a" {
		t.Errorf("expected key-a created, got %v", client.created)
	}
	if !strings.Contains(stderr.String(), "Secret key-a created") {
		t.Errorf("expected creation message in stderr, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "gcloud secrets versions add key-a") {
		t.Errorf("expected gcloud hint in stderr, got %q", stderr.String())
	}
}

func TestEnsureSecrets_Mixed(t *testing.T) {
	client := newMockClient()
	client.existing["projects/proj/secrets/exists"] = true
	// "missing" does not exist

	config := SecretConfig{
		ProjectID: "proj",
		Secrets: []SecretMount{
			{EnvVar: "EXISTS", SecretName: "exists"},
			{EnvVar: "MISSING", SecretName: "missing"},
		},
	}

	var stderr bytes.Buffer
	mounts, err := EnsureSecrets(context.Background(), client, config, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mounts) != 2 {
		t.Errorf("got %d mounts, want 2", len(mounts))
	}
	if len(client.created) != 1 || client.created[0] != "missing" {
		t.Errorf("expected only missing created, got %v", client.created)
	}
}

func TestEnsureSecrets_GetSecretError(t *testing.T) {
	client := newMockClient()
	client.getErr["projects/proj/secrets/bad"] = errors.New("permission denied")

	config := SecretConfig{
		ProjectID: "proj",
		Secrets: []SecretMount{
			{EnvVar: "BAD", SecretName: "bad"},
		},
	}

	var stderr bytes.Buffer
	_, err := EnsureSecrets(context.Background(), client, config, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "checking secret bad") {
		t.Errorf("error should mention checking secret, got %q", err)
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error should wrap original, got %q", err)
	}
}

func TestEnsureSecrets_CreateSecretError(t *testing.T) {
	client := newMockClient()
	client.createErr["fail"] = errors.New("quota exceeded")

	config := SecretConfig{
		ProjectID: "proj",
		Secrets: []SecretMount{
			{EnvVar: "FAIL", SecretName: "fail"},
		},
	}

	var stderr bytes.Buffer
	_, err := EnsureSecrets(context.Background(), client, config, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "creating secret fail") {
		t.Errorf("error should mention creating secret, got %q", err)
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("error should wrap original, got %q", err)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&NotFoundErr{Name: "x"}) {
		t.Error("expected IsNotFound to return true for NotFoundErr")
	}
	if IsNotFound(errors.New("other")) {
		t.Error("expected IsNotFound to return false for other error")
	}
	if IsNotFound(nil) {
		t.Error("expected IsNotFound to return false for nil")
	}
}
