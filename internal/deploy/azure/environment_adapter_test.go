package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"
)

type stubEnvironmentAPI struct {
	getFunc            func(ctx context.Context, rg, name string, opts *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error)
	createOrUpdateFunc func(ctx context.Context, rg, name string, envelope armappcontainers.ManagedEnvironment, opts *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error)
}

func (s *stubEnvironmentAPI) Get(ctx context.Context, rg, name string, opts *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
	return s.getFunc(ctx, rg, name, opts)
}

func (s *stubEnvironmentAPI) CreateOrUpdate(ctx context.Context, rg, name string, envelope armappcontainers.ManagedEnvironment, opts *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error) {
	return s.createOrUpdateFunc(ctx, rg, name, envelope, opts)
}

func TestEnvironmentAdapter_InterfaceCompliance(t *testing.T) {
	var _ ManagedEnvironmentClient = (*EnvironmentAdapter)(nil)
}

func TestEnvironmentAdapter_EnsureEnvironment_Exists(t *testing.T) {
	envID := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.App/managedEnvironments/myenv"
	stub := &stubEnvironmentAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
			return armappcontainers.ManagedEnvironmentsClientGetResponse{
				ManagedEnvironment: armappcontainers.ManagedEnvironment{
					ID: &envID,
				},
			}, nil
		},
	}
	adapter := &EnvironmentAdapter{client: stub}
	id, err := adapter.EnsureEnvironment(context.Background(), "rg", "myenv", "eastus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != envID {
		t.Fatalf("got ID %q, want %q", id, envID)
	}
}

func TestEnvironmentAdapter_EnsureEnvironment_NotFound_Creates(t *testing.T) {
	envID := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.App/managedEnvironments/newenv"
	stub := &stubEnvironmentAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
			return armappcontainers.ManagedEnvironmentsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ManagedEnvironment, _ *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error) {
			return &armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse{
				ManagedEnvironment: armappcontainers.ManagedEnvironment{
					ID: &envID,
				},
			}, nil
		},
	}
	adapter := &EnvironmentAdapter{client: stub}
	id, err := adapter.EnsureEnvironment(context.Background(), "rg", "newenv", "eastus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != envID {
		t.Fatalf("got ID %q, want %q", id, envID)
	}
}

func TestEnvironmentAdapter_EnsureEnvironment_GetError(t *testing.T) {
	stub := &stubEnvironmentAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
			return armappcontainers.ManagedEnvironmentsClientGetResponse{}, errors.New("access denied")
		},
	}
	adapter := &EnvironmentAdapter{client: stub}
	_, err := adapter.EnsureEnvironment(context.Background(), "rg", "myenv", "eastus")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnvironmentAdapter_EnsureEnvironment_CreateError(t *testing.T) {
	stub := &stubEnvironmentAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
			return armappcontainers.ManagedEnvironmentsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ManagedEnvironment, _ *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error) {
			return nil, errors.New("quota exceeded")
		},
	}
	adapter := &EnvironmentAdapter{client: stub}
	_, err := adapter.EnsureEnvironment(context.Background(), "rg", "myenv", "eastus")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEnvironmentAdapter_EnsureEnvironment_NilID(t *testing.T) {
	stub := &stubEnvironmentAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
			return armappcontainers.ManagedEnvironmentsClientGetResponse{
				ManagedEnvironment: armappcontainers.ManagedEnvironment{
					ID: nil,
				},
			}, nil
		},
	}
	adapter := &EnvironmentAdapter{client: stub}
	_, err := adapter.EnsureEnvironment(context.Background(), "rg", "myenv", "eastus")
	if err == nil {
		t.Fatal("expected error for nil ID, got nil")
	}
}

func TestEnvironmentAdapter_CreateEnvironment_NilID(t *testing.T) {
	stub := &stubEnvironmentAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ManagedEnvironmentsClientGetOptions) (armappcontainers.ManagedEnvironmentsClientGetResponse, error) {
			return armappcontainers.ManagedEnvironmentsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ManagedEnvironment, _ *armappcontainers.ManagedEnvironmentsClientBeginCreateOrUpdateOptions) (*armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse, error) {
			return &armappcontainers.ManagedEnvironmentsClientCreateOrUpdateResponse{
				ManagedEnvironment: armappcontainers.ManagedEnvironment{
					ID: nil,
				},
			}, nil
		},
	}
	adapter := &EnvironmentAdapter{client: stub}
	_, err := adapter.EnsureEnvironment(context.Background(), "rg", "myenv", "eastus")
	if err == nil {
		t.Fatal("expected error for nil ID on created environment, got nil")
	}
}
