package aws

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// --- High-level mock for IAMClient (used by EnsureTaskRoles tests) ---

type mockIAMClient struct {
	getRoleOut map[string]*Role
	getRoleErr map[string]error
	createOut  map[string]*Role
	createErr  error
	attachErr  error

	createCalled []string
	attachCalled []struct{ RoleName, PolicyARN string }
}

func (m *mockIAMClient) GetRole(_ context.Context, roleName string) (*Role, error) {
	if err, ok := m.getRoleErr[roleName]; ok {
		return nil, err
	}
	if role, ok := m.getRoleOut[roleName]; ok {
		return role, nil
	}
	return nil, ErrRoleNotFound
}

func (m *mockIAMClient) CreateRole(_ context.Context, input *CreateRoleInput) (*Role, error) {
	m.createCalled = append(m.createCalled, input.RoleName)
	if m.createErr != nil {
		return nil, m.createErr
	}
	if role, ok := m.createOut[input.RoleName]; ok {
		return role, nil
	}
	return &Role{ARN: "arn:aws:iam::123456789012:role/" + input.RoleName, RoleName: input.RoleName}, nil
}

func (m *mockIAMClient) AttachRolePolicy(_ context.Context, roleName, policyARN string) error {
	m.attachCalled = append(m.attachCalled, struct{ RoleName, PolicyARN string }{roleName, policyARN})
	return m.attachErr
}

func TestEnsureTaskRoles_BothExist(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{
			"mint-ecs-execution-myservice": {ARN: "arn:exec", RoleName: "mint-ecs-execution-myservice"},
			"mint-ecs-task-myservice":      {ARN: "arn:task", RoleName: "mint-ecs-task-myservice"},
		},
		getRoleErr: map[string]error{},
	}
	roles, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roles.ExecutionRoleARN != "arn:exec" {
		t.Fatalf("got ExecutionRoleARN %q, want %q", roles.ExecutionRoleARN, "arn:exec")
	}
	if roles.TaskRoleARN != "arn:task" {
		t.Fatalf("got TaskRoleARN %q, want %q", roles.TaskRoleARN, "arn:task")
	}
	if len(client.createCalled) != 0 {
		t.Fatalf("expected no CreateRole calls, got %v", client.createCalled)
	}
}

func TestEnsureTaskRoles_ExecNotFound_Creates(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{
			"mint-ecs-task-myservice": {ARN: "arn:task", RoleName: "mint-ecs-task-myservice"},
		},
		getRoleErr: map[string]error{
			"mint-ecs-execution-myservice": ErrRoleNotFound,
		},
		createOut: map[string]*Role{
			"mint-ecs-execution-myservice": {ARN: "arn:exec-new", RoleName: "mint-ecs-execution-myservice"},
		},
	}
	roles, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roles.ExecutionRoleARN != "arn:exec-new" {
		t.Fatalf("got ExecutionRoleARN %q, want %q", roles.ExecutionRoleARN, "arn:exec-new")
	}
	if len(client.createCalled) != 1 || client.createCalled[0] != "mint-ecs-execution-myservice" {
		t.Fatalf("expected CreateRole for execution role, got %v", client.createCalled)
	}
}

func TestEnsureTaskRoles_TaskNotFound_Creates(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{
			"mint-ecs-execution-myservice": {ARN: "arn:exec", RoleName: "mint-ecs-execution-myservice"},
		},
		getRoleErr: map[string]error{
			"mint-ecs-task-myservice": ErrRoleNotFound,
		},
		createOut: map[string]*Role{
			"mint-ecs-task-myservice": {ARN: "arn:task-new", RoleName: "mint-ecs-task-myservice"},
		},
	}
	roles, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roles.TaskRoleARN != "arn:task-new" {
		t.Fatalf("got TaskRoleARN %q, want %q", roles.TaskRoleARN, "arn:task-new")
	}
}

func TestEnsureTaskRoles_GetRoleUnexpectedError(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{},
		getRoleErr: map[string]error{
			"mint-ecs-execution-myservice": errors.New("access denied"),
		},
	}
	_, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureTaskRoles_TaskRoleGetUnexpectedError(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{
			"mint-ecs-execution-myservice": {ARN: "arn:exec", RoleName: "mint-ecs-execution-myservice"},
		},
		getRoleErr: map[string]error{
			"mint-ecs-task-myservice": errors.New("service unavailable"),
		},
	}
	_, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "service unavailable") {
		t.Errorf("error should contain 'service unavailable', got: %v", err)
	}
}

func TestEnsureTaskRoles_CreateRoleFails(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{},
		getRoleErr: map[string]error{
			"mint-ecs-execution-myservice": ErrRoleNotFound,
		},
		createErr: errors.New("quota exceeded"),
	}
	_, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureTaskRoles_AttachPolicyFails(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{
			"mint-ecs-execution-myservice": {ARN: "arn:exec", RoleName: "mint-ecs-execution-myservice"},
		},
		getRoleErr: map[string]error{},
		attachErr:  errors.New("policy not found"),
	}
	_, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnsureTaskRoles_TaskRoleCreateFails(t *testing.T) {
	client := &mockIAMClient{
		getRoleOut: map[string]*Role{
			"mint-ecs-execution-myservice": {ARN: "arn:exec", RoleName: "mint-ecs-execution-myservice"},
		},
		getRoleErr: map[string]error{
			"mint-ecs-task-myservice": ErrRoleNotFound,
		},
		createErr: errors.New("create failed"),
	}
	_, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ensure task role") {
		t.Errorf("error should mention task role, got: %v", err)
	}
}

func TestEcsTrustPolicy_ReturnsValidJSON(t *testing.T) {
	policy, err := ecsTrustPolicy()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]any
	if jsonErr := json.Unmarshal([]byte(policy), &parsed); jsonErr != nil {
		t.Fatalf("trust policy is not valid JSON: %v", jsonErr)
	}
	if _, ok := parsed["Version"]; !ok {
		t.Error("trust policy missing Version field")
	}
	if _, ok := parsed["Statement"]; !ok {
		t.Error("trust policy missing Statement field")
	}
}

func TestEcsTrustPolicy_MarshalError(t *testing.T) {
	orig := jsonMarshal
	defer func() { jsonMarshal = orig }()

	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("marshal boom")
	}

	_, err := ecsTrustPolicy()
	if err == nil {
		t.Fatal("expected error when jsonMarshal fails")
	}
	if !strings.Contains(err.Error(), "marshal boom") {
		t.Errorf("expected error containing 'marshal boom', got: %v", err)
	}
}

func TestEnsureTaskRoles_MarshalError(t *testing.T) {
	orig := jsonMarshal
	defer func() { jsonMarshal = orig }()

	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("marshal failure")
	}

	client := &mockIAMClient{
		getRoleOut: map[string]*Role{},
		getRoleErr: map[string]error{},
	}
	_, err := EnsureTaskRoles(context.Background(), client, "myservice")
	if err == nil {
		t.Fatal("expected error when jsonMarshal fails")
	}
	if !strings.Contains(err.Error(), "marshal trust policy") {
		t.Errorf("expected error containing 'marshal trust policy', got: %v", err)
	}
}

func TestIAMAdapterInterface(t *testing.T) {
	var _ IAMClient = (*IAMAdapter)(nil)
	var _ OIDCClient = (*IAMAdapter)(nil)
}

// --- SDK-level mock for iamAPI ---

type mockIAMAPI struct {
	getRoleFn                     func(ctx context.Context, input *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	createRoleFn                  func(ctx context.Context, input *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	attachRolePolicyFn            func(ctx context.Context, input *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error)
	getOpenIDConnectProviderFn    func(ctx context.Context, input *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error)
	createOpenIDConnectProviderFn func(ctx context.Context, input *iam.CreateOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error)
}

func (m *mockIAMAPI) GetRole(ctx context.Context, input *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
	return m.getRoleFn(ctx, input, optFns...)
}

func (m *mockIAMAPI) CreateRole(ctx context.Context, input *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
	return m.createRoleFn(ctx, input, optFns...)
}

func (m *mockIAMAPI) AttachRolePolicy(ctx context.Context, input *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
	return m.attachRolePolicyFn(ctx, input, optFns...)
}

func (m *mockIAMAPI) GetOpenIDConnectProvider(ctx context.Context, input *iam.GetOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
	return m.getOpenIDConnectProviderFn(ctx, input, optFns...)
}

func (m *mockIAMAPI) CreateOpenIDConnectProvider(ctx context.Context, input *iam.CreateOpenIDConnectProviderInput, optFns ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error) {
	return m.createOpenIDConnectProviderFn(ctx, input, optFns...)
}

func TestIAMAdapter_GetRole(t *testing.T) {
	tests := []struct {
		name     string
		sdkOut   *iam.GetRoleOutput
		sdkErr   error
		wantErr  error
		wantRole *Role
	}{
		{
			name: "success",
			sdkOut: &iam.GetRoleOutput{
				Role: &iamtypes.Role{
					Arn:      aws.String("arn:aws:iam::123456789012:role/test-role"),
					RoleName: aws.String("test-role"),
				},
			},
			wantRole: &Role{ARN: "arn:aws:iam::123456789012:role/test-role", RoleName: "test-role"},
		},
		{
			name:    "error",
			sdkErr:  errors.New("access denied"),
			wantErr: errors.New("access denied"),
		},
		{
			name:    "not found maps to ErrRoleNotFound",
			sdkErr:  &iamtypes.NoSuchEntityException{Message: aws.String("not found")},
			wantErr: ErrRoleNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				getRoleFn: func(_ context.Context, input *iam.GetRoleInput, _ ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
					if aws.ToString(input.RoleName) != "test-role" {
						t.Errorf("expected role name %q, got %q", "test-role", aws.ToString(input.RoleName))
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return tt.sdkOut, nil
				},
			}
			adapter := &IAMAdapter{client: mock}
			role, err := adapter.GetRole(context.Background(), "test-role")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(tt.wantErr, ErrRoleNotFound) {
					if !errors.Is(err, ErrRoleNotFound) {
						t.Errorf("expected ErrRoleNotFound, got: %v", err)
					}
				} else if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if role.ARN != tt.wantRole.ARN {
				t.Errorf("got ARN %q, want %q", role.ARN, tt.wantRole.ARN)
			}
			if role.RoleName != tt.wantRole.RoleName {
				t.Errorf("got RoleName %q, want %q", role.RoleName, tt.wantRole.RoleName)
			}
		})
	}
}

func TestIAMAdapter_CreateRole(t *testing.T) {
	tests := []struct {
		name    string
		sdkErr  error
		wantErr bool
	}{
		{
			name: "success",
		},
		{
			name:    "error",
			sdkErr:  errors.New("limit exceeded"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				createRoleFn: func(_ context.Context, input *iam.CreateRoleInput, _ ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
					if aws.ToString(input.RoleName) != "new-role" {
						t.Errorf("expected role name %q, got %q", "new-role", aws.ToString(input.RoleName))
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &iam.CreateRoleOutput{
						Role: &iamtypes.Role{
							Arn:      aws.String("arn:aws:iam::123456789012:role/new-role"),
							RoleName: aws.String("new-role"),
						},
					}, nil
				},
			}
			adapter := &IAMAdapter{client: mock}
			role, err := adapter.CreateRole(context.Background(), &CreateRoleInput{
				RoleName:                 "new-role",
				AssumeRolePolicyDocument: "{}",
				Description:              "test",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateRole() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if role.ARN != "arn:aws:iam::123456789012:role/new-role" {
					t.Errorf("got ARN %q, want %q", role.ARN, "arn:aws:iam::123456789012:role/new-role")
				}
			}
		})
	}
}

func TestIAMAdapter_AttachRolePolicy(t *testing.T) {
	tests := []struct {
		name    string
		sdkErr  error
		wantErr bool
	}{
		{
			name: "success",
		},
		{
			name:    "error",
			sdkErr:  errors.New("policy not found"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				attachRolePolicyFn: func(_ context.Context, input *iam.AttachRolePolicyInput, _ ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
					if aws.ToString(input.RoleName) != "my-role" {
						t.Errorf("expected role name %q, got %q", "my-role", aws.ToString(input.RoleName))
					}
					if aws.ToString(input.PolicyArn) != "arn:aws:iam::aws:policy/test" {
						t.Errorf("expected policy ARN %q, got %q", "arn:aws:iam::aws:policy/test", aws.ToString(input.PolicyArn))
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &iam.AttachRolePolicyOutput{}, nil
				},
			}
			adapter := &IAMAdapter{client: mock}
			err := adapter.AttachRolePolicy(context.Background(), "my-role", "arn:aws:iam::aws:policy/test")
			if (err != nil) != tt.wantErr {
				t.Errorf("AttachRolePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIAMAdapter_GetOpenIDConnectProvider(t *testing.T) {
	tests := []struct {
		name    string
		sdkErr  error
		wantErr error
	}{
		{
			name: "success",
		},
		{
			name:    "error",
			sdkErr:  errors.New("access denied"),
			wantErr: errors.New("access denied"),
		},
		{
			name:    "not found maps to ErrOIDCProviderNotFound",
			sdkErr:  &iamtypes.NoSuchEntityException{Message: aws.String("not found")},
			wantErr: ErrOIDCProviderNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				getOpenIDConnectProviderFn: func(_ context.Context, input *iam.GetOpenIDConnectProviderInput, _ ...func(*iam.Options)) (*iam.GetOpenIDConnectProviderOutput, error) {
					if aws.ToString(input.OpenIDConnectProviderArn) != "arn:oidc" {
						t.Errorf("expected ARN %q, got %q", "arn:oidc", aws.ToString(input.OpenIDConnectProviderArn))
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &iam.GetOpenIDConnectProviderOutput{}, nil
				},
			}
			adapter := &IAMAdapter{client: mock}
			err := adapter.GetOpenIDConnectProvider(context.Background(), "arn:oidc")
			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if errors.Is(tt.wantErr, ErrOIDCProviderNotFound) {
					if !errors.Is(err, ErrOIDCProviderNotFound) {
						t.Errorf("expected ErrOIDCProviderNotFound, got: %v", err)
					}
				} else if !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIAMAdapter_CreateOpenIDConnectProvider(t *testing.T) {
	tests := []struct {
		name    string
		sdkErr  error
		wantErr bool
		wantARN string
	}{
		{
			name:    "success",
			wantARN: "arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com",
		},
		{
			name:    "error",
			sdkErr:  errors.New("quota exceeded"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockIAMAPI{
				createOpenIDConnectProviderFn: func(_ context.Context, input *iam.CreateOpenIDConnectProviderInput, _ ...func(*iam.Options)) (*iam.CreateOpenIDConnectProviderOutput, error) {
					if aws.ToString(input.Url) != "https://token.actions.githubusercontent.com" {
						t.Errorf("expected URL %q, got %q", "https://token.actions.githubusercontent.com", aws.ToString(input.Url))
					}
					if len(input.ThumbprintList) != 1 {
						t.Errorf("expected 1 thumbprint, got %d", len(input.ThumbprintList))
					}
					if len(input.ClientIDList) != 1 || input.ClientIDList[0] != "sts.amazonaws.com" {
						t.Errorf("expected client ID list [sts.amazonaws.com], got %v", input.ClientIDList)
					}
					if tt.sdkErr != nil {
						return nil, tt.sdkErr
					}
					return &iam.CreateOpenIDConnectProviderOutput{
						OpenIDConnectProviderArn: aws.String("arn:aws:iam::123456789012:oidc-provider/token.actions.githubusercontent.com"),
					}, nil
				},
			}
			adapter := &IAMAdapter{client: mock}
			arn, err := adapter.CreateOpenIDConnectProvider(context.Background(), "https://token.actions.githubusercontent.com", []string{"thumb1"})
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOpenIDConnectProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && arn != tt.wantARN {
				t.Errorf("got ARN %q, want %q", arn, tt.wantARN)
			}
		})
	}
}

func TestNewIAMAdapter(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	adapter := NewIAMAdapter(cfg)
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.client == nil {
		t.Fatal("expected non-nil client")
	}
}
