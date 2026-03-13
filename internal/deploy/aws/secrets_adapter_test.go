package aws

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type mockSecretsClient struct {
	describeOut map[string]*SecretInfo
	describeErr map[string]error
	createOut   map[string]*SecretInfo
	createErr   error

	createCalled []string
}

func (m *mockSecretsClient) DescribeSecret(_ context.Context, secretID string) (*SecretInfo, error) {
	if err, ok := m.describeErr[secretID]; ok {
		return nil, err
	}
	if info, ok := m.describeOut[secretID]; ok {
		return info, nil
	}
	return nil, ErrSecretNotFound
}

func (m *mockSecretsClient) CreateSecret(_ context.Context, input *CreateSecretInput) (*SecretInfo, error) {
	m.createCalled = append(m.createCalled, input.Name)
	if m.createErr != nil {
		return nil, m.createErr
	}
	if info, ok := m.createOut[input.Name]; ok {
		return info, nil
	}
	return &SecretInfo{ARN: "arn:aws:secretsmanager:us-east-1:123456789012:secret:" + input.Name, Name: input.Name}, nil
}

func (m *mockSecretsClient) GetSecretValue(_ context.Context, secretID string) (string, error) {
	return "", nil
}

func TestEnsureSecrets_AllExist(t *testing.T) {
	client := &mockSecretsClient{
		describeOut: map[string]*SecretInfo{
			"my-api-key":     {ARN: "arn:key", Name: "my-api-key"},
			"my-db-password": {ARN: "arn:db", Name: "my-db-password"},
		},
		describeErr: map[string]error{},
	}
	var buf bytes.Buffer
	arns, err := EnsureSecrets(context.Background(), client, map[string]string{
		"API_KEY":     "my-api-key",
		"DB_PASSWORD": "my-db-password",
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(arns) != 2 {
		t.Fatalf("expected 2 ARNs, got %d", len(arns))
	}
	if len(client.createCalled) != 0 {
		t.Fatalf("expected no CreateSecret calls, got %v", client.createCalled)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", buf.String())
	}
}

func TestEnsureSecrets_SomeNotFound_Creates(t *testing.T) {
	client := &mockSecretsClient{
		describeOut: map[string]*SecretInfo{
			"existing-secret": {ARN: "arn:existing", Name: "existing-secret"},
		},
		describeErr: map[string]error{
			"new-secret": ErrSecretNotFound,
		},
		createOut: map[string]*SecretInfo{
			"new-secret": {ARN: "arn:new", Name: "new-secret"},
		},
	}
	var buf bytes.Buffer
	arns, err := EnsureSecrets(context.Background(), client, map[string]string{
		"EXISTING": "existing-secret",
		"NEW":      "new-secret",
	}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(arns) != 2 {
		t.Fatalf("expected 2 ARNs, got %d", len(arns))
	}
	if len(client.createCalled) != 1 || client.createCalled[0] != "new-secret" {
		t.Fatalf("expected CreateSecret for new-secret, got %v", client.createCalled)
	}
	if buf.Len() == 0 {
		t.Fatal("expected stderr output for newly created secret")
	}
}

func TestEnsureSecrets_DescribeUnexpectedError(t *testing.T) {
	client := &mockSecretsClient{
		describeOut: map[string]*SecretInfo{},
		describeErr: map[string]error{
			"bad-secret": errors.New("access denied"),
		},
	}
	var buf bytes.Buffer
	_, err := EnsureSecrets(context.Background(), client, map[string]string{
		"BAD": "bad-secret",
	}, &buf)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureSecrets_CreateFails(t *testing.T) {
	client := &mockSecretsClient{
		describeOut: map[string]*SecretInfo{},
		describeErr: map[string]error{
			"fail-secret": ErrSecretNotFound,
		},
		createErr: errors.New("quota exceeded"),
	}
	var buf bytes.Buffer
	_, err := EnsureSecrets(context.Background(), client, map[string]string{
		"FAIL": "fail-secret",
	}, &buf)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSecretsManagerAdapterInterface(t *testing.T) {
	var _ SecretsClient = (*SecretsManagerAdapter)(nil)
}
