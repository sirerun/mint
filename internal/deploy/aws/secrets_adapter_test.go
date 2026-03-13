package aws

import (
	"bytes"
	"context"
	"errors"
	"testing"

	sdkaws "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
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

// --- secretsManagerAPI mock for adapter-level tests ---

type stubSecretsAPI struct {
	describeFn func(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
	createFn   func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	getValueFn func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func (s *stubSecretsAPI) DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
	return s.describeFn(ctx, params, optFns...)
}

func (s *stubSecretsAPI) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	return s.createFn(ctx, params, optFns...)
}

func (s *stubSecretsAPI) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return s.getValueFn(ctx, params, optFns...)
}

func sPtr(s string) *string { return &s }

func TestNewSecretsManagerAdapter(t *testing.T) {
	adapter := NewSecretsManagerAdapter(sdkaws.Config{Region: "us-east-1"})
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestSecretsManagerAdapter_DescribeSecret(t *testing.T) {
	arn := "arn:aws:secretsmanager:us-east-1:123456:secret:my-secret"
	name := "my-secret"

	tests := []struct {
		name    string
		stubFn  func(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
		wantErr error
		isOther bool
		wantARN string
	}{
		{
			name: "success",
			stubFn: func(_ context.Context, _ *secretsmanager.DescribeSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
				return &secretsmanager.DescribeSecretOutput{
					ARN:  &arn,
					Name: &name,
				}, nil
			},
			wantARN: arn,
		},
		{
			name: "not found mapped to sentinel",
			stubFn: func(_ context.Context, _ *secretsmanager.DescribeSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
				return nil, &smtypes.ResourceNotFoundException{Message: sPtr("not found")}
			},
			wantErr: ErrSecretNotFound,
		},
		{
			name: "other error passed through",
			stubFn: func(_ context.Context, _ *secretsmanager.DescribeSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error) {
				return nil, errors.New("access denied")
			},
			isOther: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &SecretsManagerAdapter{client: &stubSecretsAPI{describeFn: tt.stubFn}}
			out, err := adapter.DescribeSecret(context.Background(), "my-secret")

			if tt.isOther {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(err, ErrSecretNotFound) {
					t.Fatal("expected non-sentinel error")
				}
				return
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.ARN != tt.wantARN {
				t.Fatalf("expected ARN %q, got %q", tt.wantARN, out.ARN)
			}
		})
	}
}

func TestSecretsManagerAdapter_CreateSecret(t *testing.T) {
	arn := "arn:aws:secretsmanager:us-east-1:123456:secret:new-secret"
	name := "new-secret"

	tests := []struct {
		name    string
		stubFn  func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
		wantErr bool
		wantARN string
	}{
		{
			name: "success",
			stubFn: func(_ context.Context, _ *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return &secretsmanager.CreateSecretOutput{
					ARN:  &arn,
					Name: &name,
				}, nil
			},
			wantARN: arn,
		},
		{
			name: "error",
			stubFn: func(_ context.Context, _ *secretsmanager.CreateSecretInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
				return nil, errors.New("quota exceeded")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &SecretsManagerAdapter{client: &stubSecretsAPI{createFn: tt.stubFn}}
			out, err := adapter.CreateSecret(context.Background(), &CreateSecretInput{
				Name:        "new-secret",
				Description: "test",
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.ARN != tt.wantARN {
				t.Fatalf("expected ARN %q, got %q", tt.wantARN, out.ARN)
			}
		})
	}
}

func TestSecretsManagerAdapter_GetSecretValue(t *testing.T) {
	secret := "super-secret-value"

	tests := []struct {
		name    string
		stubFn  func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
		wantErr error
		isOther bool
		wantVal string
	}{
		{
			name: "success",
			stubFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: &secret,
				}, nil
			},
			wantVal: secret,
		},
		{
			name: "not found mapped to sentinel",
			stubFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, &smtypes.ResourceNotFoundException{Message: sPtr("not found")}
			},
			wantErr: ErrSecretNotFound,
		},
		{
			name: "other error passed through",
			stubFn: func(_ context.Context, _ *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, errors.New("throttled")
			},
			isOther: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &SecretsManagerAdapter{client: &stubSecretsAPI{getValueFn: tt.stubFn}}
			val, err := adapter.GetSecretValue(context.Background(), "my-secret")

			if tt.isOther {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(err, ErrSecretNotFound) {
					t.Fatal("expected non-sentinel error")
				}
				return
			}

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if val != tt.wantVal {
				t.Fatalf("expected value %q, got %q", tt.wantVal, val)
			}
		})
	}
}
