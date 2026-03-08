package gcp

import (
	"context"
	"errors"
	"testing"
)

// mockCloudRunClient implements CloudRunClient for testing.
type mockCloudRunClient struct {
	getFunc    func(ctx context.Context, name string) (*Service, error)
	createFunc func(ctx context.Context, config *ServiceConfig) (*Service, error)
	updateFunc func(ctx context.Context, config *ServiceConfig) (*Service, error)
}

func (m *mockCloudRunClient) GetService(ctx context.Context, name string) (*Service, error) {
	return m.getFunc(ctx, name)
}

func (m *mockCloudRunClient) CreateService(ctx context.Context, config *ServiceConfig) (*Service, error) {
	return m.createFunc(ctx, config)
}

func (m *mockCloudRunClient) UpdateService(ctx context.Context, config *ServiceConfig) (*Service, error) {
	return m.updateFunc(ctx, config)
}

func validConfig() *ServiceConfig {
	return &ServiceConfig{
		ProjectID:   "my-project",
		Region:      "us-central1",
		ServiceName: "my-service",
		ImageURI:    "gcr.io/my-project/my-image:latest",
		Port:        8080,
	}
}

func TestServiceFullName(t *testing.T) {
	got := ServiceFullName("proj", "us-east1", "svc")
	want := "projects/proj/locations/us-east1/services/svc"
	if got != want {
		t.Errorf("ServiceFullName() = %q, want %q", got, want)
	}
}

func TestEnsureService_CreatesWhenNotFound(t *testing.T) {
	created := &Service{
		Name:         "projects/my-project/locations/us-central1/services/my-service",
		URL:          "https://my-service-abc123.a.run.app",
		RevisionName: "my-service-00001",
		Status:       "Ready",
	}

	client := &mockCloudRunClient{
		getFunc: func(_ context.Context, _ string) (*Service, error) {
			return nil, ErrNotFound
		},
		createFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			return created, nil
		},
		updateFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("UpdateService should not be called when service does not exist")
			return nil, nil
		},
	}

	svc, err := EnsureService(context.Background(), client, validConfig())
	if err != nil {
		t.Fatalf("EnsureService() error = %v", err)
	}
	if svc.URL != created.URL {
		t.Errorf("URL = %q, want %q", svc.URL, created.URL)
	}
	if svc.RevisionName != created.RevisionName {
		t.Errorf("RevisionName = %q, want %q", svc.RevisionName, created.RevisionName)
	}
}

func TestEnsureService_UpdatesWhenExists(t *testing.T) {
	existing := &Service{
		Name:         "projects/my-project/locations/us-central1/services/my-service",
		URL:          "https://my-service-abc123.a.run.app",
		RevisionName: "my-service-00001",
		Status:       "Ready",
	}
	updated := &Service{
		Name:         existing.Name,
		URL:          existing.URL,
		RevisionName: "my-service-00002",
		Status:       "Ready",
	}

	client := &mockCloudRunClient{
		getFunc: func(_ context.Context, _ string) (*Service, error) {
			return existing, nil
		},
		createFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("CreateService should not be called when service exists")
			return nil, nil
		},
		updateFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			return updated, nil
		},
	}

	svc, err := EnsureService(context.Background(), client, validConfig())
	if err != nil {
		t.Fatalf("EnsureService() error = %v", err)
	}
	if svc.RevisionName != updated.RevisionName {
		t.Errorf("RevisionName = %q, want %q", svc.RevisionName, updated.RevisionName)
	}
}

func TestEnsureService_GetServiceError(t *testing.T) {
	client := &mockCloudRunClient{
		getFunc: func(_ context.Context, _ string) (*Service, error) {
			return nil, errors.New("permission denied")
		},
		createFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("CreateService should not be called on unexpected error")
			return nil, nil
		},
		updateFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("UpdateService should not be called on unexpected error")
			return nil, nil
		},
	}

	_, err := EnsureService(context.Background(), client, validConfig())
	if err == nil {
		t.Fatal("EnsureService() expected error, got nil")
	}
}

func TestEnsureService_CreateServiceError(t *testing.T) {
	client := &mockCloudRunClient{
		getFunc: func(_ context.Context, _ string) (*Service, error) {
			return nil, ErrNotFound
		},
		createFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			return nil, errors.New("quota exceeded")
		},
		updateFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("UpdateService should not be called")
			return nil, nil
		},
	}

	_, err := EnsureService(context.Background(), client, validConfig())
	if err == nil {
		t.Fatal("EnsureService() expected error, got nil")
	}
}

func TestEnsureService_UpdateServiceError(t *testing.T) {
	client := &mockCloudRunClient{
		getFunc: func(_ context.Context, _ string) (*Service, error) {
			return &Service{Name: "existing"}, nil
		},
		createFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("CreateService should not be called")
			return nil, nil
		},
		updateFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			return nil, errors.New("update failed")
		},
	}

	_, err := EnsureService(context.Background(), client, validConfig())
	if err == nil {
		t.Fatal("EnsureService() expected error, got nil")
	}
}

func TestEnsureService_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *ServiceConfig
	}{
		{"nil config", nil},
		{"empty ProjectID", &ServiceConfig{Region: "r", ServiceName: "s", ImageURI: "i"}},
		{"empty Region", &ServiceConfig{ProjectID: "p", ServiceName: "s", ImageURI: "i"}},
		{"empty ServiceName", &ServiceConfig{ProjectID: "p", Region: "r", ImageURI: "i"}},
		{"empty ImageURI", &ServiceConfig{ProjectID: "p", Region: "r", ServiceName: "s"}},
	}

	client := &mockCloudRunClient{
		getFunc: func(_ context.Context, _ string) (*Service, error) {
			t.Fatal("GetService should not be called with invalid config")
			return nil, nil
		},
		createFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("CreateService should not be called with invalid config")
			return nil, nil
		},
		updateFunc: func(_ context.Context, _ *ServiceConfig) (*Service, error) {
			t.Fatal("UpdateService should not be called with invalid config")
			return nil, nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EnsureService(context.Background(), client, tt.config)
			if err == nil {
				t.Fatal("EnsureService() expected error for invalid config, got nil")
			}
		})
	}
}
