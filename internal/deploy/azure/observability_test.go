package azure

import (
	"context"
	"errors"
	"testing"
)

type mockLogAnalyticsAPI struct {
	ensureWorkspaceFn func(ctx context.Context, resourceGroup, workspaceName string) (string, error)
	linkEnvironmentFn func(ctx context.Context, resourceGroup, workspaceName, environmentName string) error
}

func (m *mockLogAnalyticsAPI) EnsureWorkspace(ctx context.Context, resourceGroup, workspaceName string) (string, error) {
	return m.ensureWorkspaceFn(ctx, resourceGroup, workspaceName)
}

func (m *mockLogAnalyticsAPI) LinkEnvironment(ctx context.Context, resourceGroup, workspaceName, environmentName string) error {
	return m.linkEnvironmentFn(ctx, resourceGroup, workspaceName, environmentName)
}

func defaultLogAnalyticsMock() *mockLogAnalyticsAPI {
	return &mockLogAnalyticsAPI{
		ensureWorkspaceFn: func(_ context.Context, _, workspaceName string) (string, error) {
			return "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/" + workspaceName, nil
		},
		linkEnvironmentFn: func(_ context.Context, _, _, _ string) error {
			return nil
		},
	}
}

func TestConfigureObservability_DefaultWorkspaceName(t *testing.T) {
	var gotWorkspace string
	var gotEnvName string

	mock := defaultLogAnalyticsMock()
	mock.ensureWorkspaceFn = func(_ context.Context, _, workspaceName string) (string, error) {
		gotWorkspace = workspaceName
		return "workspace-id", nil
	}
	mock.linkEnvironmentFn = func(_ context.Context, _, _, envName string) error {
		gotEnvName = envName
		return nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ResourceGroup: "my-rg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotWorkspace != "my-svc-logs" {
		t.Errorf("workspace = %q, want %q", gotWorkspace, "my-svc-logs")
	}
	if gotEnvName != "my-svc" {
		t.Errorf("environment = %q, want %q", gotEnvName, "my-svc")
	}
}

func TestConfigureObservability_CustomWorkspaceName(t *testing.T) {
	var gotWorkspace string

	mock := defaultLogAnalyticsMock()
	mock.ensureWorkspaceFn = func(_ context.Context, _, workspaceName string) (string, error) {
		gotWorkspace = workspaceName
		return "workspace-id", nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ResourceGroup: "my-rg",
		WorkspaceName: "custom-workspace",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotWorkspace != "custom-workspace" {
		t.Errorf("workspace = %q, want %q", gotWorkspace, "custom-workspace")
	}
}

func TestConfigureObservability_ResourceGroupPassed(t *testing.T) {
	var gotRG string

	mock := defaultLogAnalyticsMock()
	mock.ensureWorkspaceFn = func(_ context.Context, rg, _ string) (string, error) {
		gotRG = rg
		return "id", nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ResourceGroup: "prod-rg",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotRG != "prod-rg" {
		t.Errorf("resource group = %q, want %q", gotRG, "prod-rg")
	}
}

func TestConfigureObservability_EnsureWorkspaceError(t *testing.T) {
	mock := defaultLogAnalyticsMock()
	mock.ensureWorkspaceFn = func(_ context.Context, _, _ string) (string, error) {
		return "", errors.New("forbidden")
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ResourceGroup: "my-rg",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "observability: ensure workspace: forbidden" {
		t.Errorf("error = %q, want %q", got, "observability: ensure workspace: forbidden")
	}
}

func TestConfigureObservability_LinkEnvironmentError(t *testing.T) {
	mock := defaultLogAnalyticsMock()
	mock.linkEnvironmentFn = func(_ context.Context, _, _, _ string) error {
		return errors.New("not found")
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ResourceGroup: "my-rg",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "observability: link environment: not found" {
		t.Errorf("error = %q, want %q", got, "observability: link environment: not found")
	}
}

func TestConfigureObservability_MissingServiceName(t *testing.T) {
	adapter := NewObservabilityAdapter(defaultLogAnalyticsMock())
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ResourceGroup: "my-rg",
	})
	if err == nil {
		t.Fatal("expected error for missing service name")
	}
}

func TestConfigureObservability_MissingResourceGroup(t *testing.T) {
	adapter := NewObservabilityAdapter(defaultLogAnalyticsMock())
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
	})
	if err == nil {
		t.Fatal("expected error for missing resource group")
	}
}
