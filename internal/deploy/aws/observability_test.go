package aws

import (
	"context"
	"errors"
	"testing"
)

type mockObservabilityAPI struct {
	createLogGroupFn        func(ctx context.Context, logGroupName string) error
	putRetentionPolicyFn    func(ctx context.Context, logGroupName string, retentionDays int) error
	updateClusterSettingsFn func(ctx context.Context, clusterName string, containerInsights bool) error
}

func (m *mockObservabilityAPI) CreateLogGroup(ctx context.Context, logGroupName string) error {
	return m.createLogGroupFn(ctx, logGroupName)
}

func (m *mockObservabilityAPI) PutRetentionPolicy(ctx context.Context, logGroupName string, retentionDays int) error {
	return m.putRetentionPolicyFn(ctx, logGroupName, retentionDays)
}

func (m *mockObservabilityAPI) UpdateClusterSettings(ctx context.Context, clusterName string, containerInsights bool) error {
	return m.updateClusterSettingsFn(ctx, clusterName, containerInsights)
}

func defaultObservabilityMock() *mockObservabilityAPI {
	return &mockObservabilityAPI{
		createLogGroupFn: func(_ context.Context, _ string) error {
			return nil
		},
		putRetentionPolicyFn: func(_ context.Context, _ string, _ int) error {
			return nil
		},
		updateClusterSettingsFn: func(_ context.Context, _ string, _ bool) error {
			return nil
		},
	}
}

func TestConfigureObservability_DefaultLogGroup(t *testing.T) {
	var gotLogGroup string
	var gotRetention int

	mock := defaultObservabilityMock()
	mock.createLogGroupFn = func(_ context.Context, logGroupName string) error {
		gotLogGroup = logGroupName
		return nil
	}
	mock.putRetentionPolicyFn = func(_ context.Context, logGroupName string, days int) error {
		gotRetention = days
		return nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
		ClusterName: "my-cluster",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotLogGroup != "/ecs/my-svc" {
		t.Errorf("log group = %q, want %q", gotLogGroup, "/ecs/my-svc")
	}
	if gotRetention != 30 {
		t.Errorf("retention = %d, want %d", gotRetention, 30)
	}
}

func TestConfigureObservability_CustomLogGroupPrefix(t *testing.T) {
	var gotLogGroup string

	mock := defaultObservabilityMock()
	mock.createLogGroupFn = func(_ context.Context, logGroupName string) error {
		gotLogGroup = logGroupName
		return nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:    "my-svc",
		ClusterName:    "my-cluster",
		LogGroupPrefix: "/custom/prefix",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotLogGroup != "/custom/prefix/my-svc" {
		t.Errorf("log group = %q, want %q", gotLogGroup, "/custom/prefix/my-svc")
	}
}

func TestConfigureObservability_EnableMetrics(t *testing.T) {
	var gotCluster string
	var gotInsights bool

	mock := defaultObservabilityMock()
	mock.updateClusterSettingsFn = func(_ context.Context, cluster string, insights bool) error {
		gotCluster = cluster
		gotInsights = insights
		return nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ClusterName:   "my-cluster",
		EnableMetrics: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotCluster != "my-cluster" {
		t.Errorf("cluster = %q, want %q", gotCluster, "my-cluster")
	}
	if !gotInsights {
		t.Error("expected container insights to be enabled")
	}
}

func TestConfigureObservability_MetricsNotCalledWhenDisabled(t *testing.T) {
	called := false

	mock := defaultObservabilityMock()
	mock.updateClusterSettingsFn = func(_ context.Context, _ string, _ bool) error {
		called = true
		return nil
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ClusterName:   "my-cluster",
		EnableMetrics: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called {
		t.Error("UpdateClusterSettings should not be called when EnableMetrics is false")
	}
}

func TestConfigureObservability_CreateLogGroupError(t *testing.T) {
	mock := defaultObservabilityMock()
	mock.createLogGroupFn = func(_ context.Context, _ string) error {
		return errors.New("access denied")
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
		ClusterName: "my-cluster",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "observability: create log group: access denied" {
		t.Errorf("error = %q, want %q", got, "observability: create log group: access denied")
	}
}

func TestConfigureObservability_RetentionPolicyError(t *testing.T) {
	mock := defaultObservabilityMock()
	mock.putRetentionPolicyFn = func(_ context.Context, _ string, _ int) error {
		return errors.New("throttled")
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
		ClusterName: "my-cluster",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "observability: set retention policy: throttled" {
		t.Errorf("error = %q, want %q", got, "observability: set retention policy: throttled")
	}
}

func TestConfigureObservability_ContainerInsightsError(t *testing.T) {
	mock := defaultObservabilityMock()
	mock.updateClusterSettingsFn = func(_ context.Context, _ string, _ bool) error {
		return errors.New("cluster not found")
	}

	adapter := NewObservabilityAdapter(mock)
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName:   "my-svc",
		ClusterName:   "my-cluster",
		EnableMetrics: true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "observability: enable container insights: cluster not found" {
		t.Errorf("error = %q, want %q", got, "observability: enable container insights: cluster not found")
	}
}

func TestConfigureObservability_MissingServiceName(t *testing.T) {
	adapter := NewObservabilityAdapter(defaultObservabilityMock())
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ClusterName: "my-cluster",
	})
	if err == nil {
		t.Fatal("expected error for missing service name")
	}
}

func TestConfigureObservability_MissingClusterName(t *testing.T) {
	adapter := NewObservabilityAdapter(defaultObservabilityMock())
	err := adapter.ConfigureObservability(context.Background(), ObservabilityConfig{
		ServiceName: "my-svc",
	})
	if err == nil {
		t.Fatal("expected error for missing cluster name")
	}
}
