package gcp

import (
	"context"
	"errors"
	"testing"
)

// mockIAMPolicyClient implements IAMPolicyClient for testing.
type mockIAMPolicyClient struct {
	policy   *IAMPolicy
	getErr   error
	setErr   error
	setCalls int
}

func (m *mockIAMPolicyClient) GetIAMPolicy(_ context.Context, _ string) (*IAMPolicy, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	// Return a copy to avoid shared mutation issues.
	cp := &IAMPolicy{
		Bindings: make([]IAMBinding, len(m.policy.Bindings)),
	}
	for i, b := range m.policy.Bindings {
		cp.Bindings[i] = IAMBinding{
			Role:    b.Role,
			Members: append([]string(nil), b.Members...),
		}
	}
	return cp, nil
}

func (m *mockIAMPolicyClient) SetIAMPolicy(_ context.Context, _ string, policy *IAMPolicy) error {
	m.setCalls++
	if m.setErr != nil {
		return m.setErr
	}
	m.policy = policy
	return nil
}

func TestConfigureIAMPolicy_PrivateRemovesAllUsers(t *testing.T) {
	client := &mockIAMPolicyClient{
		policy: &IAMPolicy{
			Bindings: []IAMBinding{
				{
					Role:    "roles/run.invoker",
					Members: []string{"allUsers", "serviceAccount:sa@proj.iam.gserviceaccount.com"},
				},
			},
		},
	}
	config := ServiceAccountConfig{
		ProjectID:   "my-project",
		ServiceName: "api",
		Public:      false,
	}

	err := ConfigureIAMPolicy(context.Background(), client, config, "projects/my-project/locations/us-central1/services/api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(client.policy.Bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(client.policy.Bindings))
	}
	for _, m := range client.policy.Bindings[0].Members {
		if m == "allUsers" || m == "allAuthenticatedUsers" {
			t.Errorf("expected public members to be removed, found %q", m)
		}
	}
	if len(client.policy.Bindings[0].Members) != 1 {
		t.Errorf("expected 1 member remaining, got %d", len(client.policy.Bindings[0].Members))
	}
}

func TestConfigureIAMPolicy_PublicAddsAllUsers(t *testing.T) {
	client := &mockIAMPolicyClient{
		policy: &IAMPolicy{
			Bindings: []IAMBinding{},
		},
	}
	config := ServiceAccountConfig{
		ProjectID:   "my-project",
		ServiceName: "api",
		Public:      true,
	}

	err := ConfigureIAMPolicy(context.Background(), client, config, "projects/my-project/locations/us-central1/services/api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, b := range client.policy.Bindings {
		if b.Role == "roles/run.invoker" {
			for _, m := range b.Members {
				if m == "allUsers" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected allUsers in roles/run.invoker binding")
	}
}

func TestConfigureIAMPolicy_Idempotent(t *testing.T) {
	// Public: allUsers already present.
	client := &mockIAMPolicyClient{
		policy: &IAMPolicy{
			Bindings: []IAMBinding{
				{
					Role:    "roles/run.invoker",
					Members: []string{"allUsers"},
				},
			},
		},
	}
	config := ServiceAccountConfig{
		ProjectID:   "my-project",
		ServiceName: "api",
		Public:      true,
	}

	err := ConfigureIAMPolicy(context.Background(), client, config, "svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := 0
	for _, b := range client.policy.Bindings {
		if b.Role == "roles/run.invoker" {
			for _, m := range b.Members {
				if m == "allUsers" {
					count++
				}
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 allUsers member, got %d", count)
	}

	// Private: no public members to remove.
	client2 := &mockIAMPolicyClient{
		policy: &IAMPolicy{
			Bindings: []IAMBinding{
				{
					Role:    "roles/run.invoker",
					Members: []string{"serviceAccount:sa@proj.iam.gserviceaccount.com"},
				},
			},
		},
	}
	config2 := ServiceAccountConfig{
		ProjectID:   "my-project",
		ServiceName: "api",
		Public:      false,
	}

	err = ConfigureIAMPolicy(context.Background(), client2, config2, "svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(client2.policy.Bindings[0].Members) != 1 {
		t.Errorf("expected 1 member, got %d", len(client2.policy.Bindings[0].Members))
	}
	if client2.policy.Bindings[0].Members[0] != "serviceAccount:sa@proj.iam.gserviceaccount.com" {
		t.Errorf("unexpected member: %s", client2.policy.Bindings[0].Members[0])
	}
}

func TestConfigureIAMPolicy_GetError(t *testing.T) {
	client := &mockIAMPolicyClient{
		getErr: errors.New("permission denied"),
	}
	config := ServiceAccountConfig{
		ProjectID:   "my-project",
		ServiceName: "api",
		Public:      false,
	}

	err := ConfigureIAMPolicy(context.Background(), client, config, "svc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, client.getErr) {
		t.Errorf("expected wrapped permission denied error, got: %v", err)
	}
}

func TestConfigureIAMPolicy_SetError(t *testing.T) {
	client := &mockIAMPolicyClient{
		policy: &IAMPolicy{Bindings: []IAMBinding{}},
		setErr: errors.New("quota exceeded"),
	}
	config := ServiceAccountConfig{
		ProjectID:   "my-project",
		ServiceName: "api",
		Public:      false,
	}

	err := ConfigureIAMPolicy(context.Background(), client, config, "svc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, client.setErr) {
		t.Errorf("expected wrapped quota exceeded error, got: %v", err)
	}
}

func TestServiceAccountEmail(t *testing.T) {
	tests := []struct {
		projectID   string
		serviceName string
		want        string
	}{
		{"my-project", "api", "mint-mcp-api@my-project.iam.gserviceaccount.com"},
		{"prod-123", "petstore", "mint-mcp-petstore@prod-123.iam.gserviceaccount.com"},
	}
	for _, tt := range tests {
		got := ServiceAccountEmail(tt.projectID, tt.serviceName)
		if got != tt.want {
			t.Errorf("ServiceAccountEmail(%q, %q) = %q, want %q", tt.projectID, tt.serviceName, got, tt.want)
		}
	}
}
