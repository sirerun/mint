package aws

import (
	"context"
	"errors"
	"testing"
)

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

func TestIAMAdapterInterface(t *testing.T) {
	var _ IAMClient = (*IAMAdapter)(nil)
}

func TestIAMAdapterOIDCInterface(t *testing.T) {
	var _ OIDCClient = (*IAMAdapter)(nil)
}
