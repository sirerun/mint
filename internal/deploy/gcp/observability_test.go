package gcp

import (
	"context"
	"errors"
	"testing"
)

type mockLoggingAPI struct {
	updateServiceLabelsFn func(ctx context.Context, projectID, region, serviceName string, labels map[string]string) error
}

func (m *mockLoggingAPI) UpdateServiceLabels(ctx context.Context, projectID, region, serviceName string, labels map[string]string) error {
	return m.updateServiceLabelsFn(ctx, projectID, region, serviceName, labels)
}

func TestConfigureObservability_Success(t *testing.T) {
	var gotLabels map[string]string
	var gotProject, gotRegion, gotService string

	mock := &mockLoggingAPI{
		updateServiceLabelsFn: func(_ context.Context, projectID, region, serviceName string, labels map[string]string) error {
			gotProject = projectID
			gotRegion = region
			gotService = serviceName
			gotLabels = labels
			return nil
		},
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ProjectID:     "my-project",
		Region:        "us-central1",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotProject != "my-project" {
		t.Errorf("project = %q, want %q", gotProject, "my-project")
	}
	if gotRegion != "us-central1" {
		t.Errorf("region = %q, want %q", gotRegion, "us-central1")
	}
	if gotService != "my-svc" {
		t.Errorf("service = %q, want %q", gotService, "my-svc")
	}
	if gotLabels["observability"] != "enabled" {
		t.Errorf("observability label = %q, want %q", gotLabels["observability"], "enabled")
	}
	if gotLabels["service"] != "my-svc" {
		t.Errorf("service label = %q, want %q", gotLabels["service"], "my-svc")
	}
	if _, ok := gotLabels["metrics"]; ok {
		t.Error("metrics label should not be set when EnableMetrics is false")
	}
}

func TestConfigureObservability_WithMetrics(t *testing.T) {
	var gotLabels map[string]string

	mock := &mockLoggingAPI{
		updateServiceLabelsFn: func(_ context.Context, _, _, _ string, labels map[string]string) error {
			gotLabels = labels
			return nil
		},
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ProjectID:     "my-project",
		Region:        "us-central1",
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotLabels["metrics"] != "enabled" {
		t.Errorf("metrics label = %q, want %q", gotLabels["metrics"], "enabled")
	}
}

func TestConfigureObservability_ClientError(t *testing.T) {
	mock := &mockLoggingAPI{
		updateServiceLabelsFn: func(_ context.Context, _, _, _ string, _ map[string]string) error {
			return errors.New("api failure")
		},
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ProjectID:     "my-project",
		Region:        "us-central1",
		EnableMetrics: false,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "observability: update labels: api failure" {
		t.Errorf("error = %q, want %q", got, "observability: update labels: api failure")
	}
}

func TestConfigureObservability_MissingServiceName(t *testing.T) {
	adapter := NewObservabilityAdapter(&mockLoggingAPI{})
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ProjectID: "my-project",
		Region:    "us-central1",
	})
	if err == nil {
		t.Fatal("expected error for missing service name")
	}
}

func TestConfigureObservability_MissingProjectID(t *testing.T) {
	adapter := NewObservabilityAdapter(&mockLoggingAPI{})
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
		Region:      "us-central1",
	})
	if err == nil {
		t.Fatal("expected error for missing project ID")
	}
}

func TestConfigureObservability_MissingRegion(t *testing.T) {
	adapter := NewObservabilityAdapter(&mockLoggingAPI{})
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
		ProjectID:   "my-project",
	})
	if err == nil {
		t.Fatal("expected error for missing region")
	}
}

func TestConfigureObservability_SanitizesServiceLabel(t *testing.T) {
	var gotLabels map[string]string

	mock := &mockLoggingAPI{
		updateServiceLabelsFn: func(_ context.Context, _, _, _ string, labels map[string]string) error {
			gotLabels = labels
			return nil
		},
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "My-Service.v2",
		ProjectID:     "my-project",
		Region:        "us-central1",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotLabels["service"] != "my-servicev2" {
		t.Errorf("service label = %q, want %q", gotLabels["service"], "my-servicev2")
	}
}
