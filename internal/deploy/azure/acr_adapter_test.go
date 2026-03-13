package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
)

type stubACRAPI struct {
	getFunc    func(ctx context.Context, rg, name string, opts *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error)
	createFunc func(ctx context.Context, rg, name string, reg armcontainerregistry.Registry, opts *armcontainerregistry.RegistriesClientBeginCreateOptions) (*armcontainerregistry.RegistriesClientCreateResponse, error)
}

func (s *stubACRAPI) Get(ctx context.Context, rg, name string, opts *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
	return s.getFunc(ctx, rg, name, opts)
}

func (s *stubACRAPI) Create(ctx context.Context, rg, name string, reg armcontainerregistry.Registry, opts *armcontainerregistry.RegistriesClientBeginCreateOptions) (*armcontainerregistry.RegistriesClientCreateResponse, error) {
	return s.createFunc(ctx, rg, name, reg, opts)
}

func TestACRAdapter_InterfaceCompliance(t *testing.T) {
	var _ ACRClient = (*ACRAdapter)(nil)
}

func TestACRAdapter_EnsureRepository_Exists(t *testing.T) {
	loginServer := "myregistry.azurecr.io"
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{
				Registry: armcontainerregistry.Registry{
					Properties: &armcontainerregistry.RegistryProperties{
						LoginServer: &loginServer,
					},
				},
			}, nil
		},
	}
	adapter := &ACRAdapter{client: stub}
	uri, err := adapter.EnsureRepository(context.Background(), "rg", "myregistry", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "myregistry.azurecr.io/myrepo"
	if uri != want {
		t.Fatalf("got URI %q, want %q", uri, want)
	}
}

func TestACRAdapter_EnsureRepository_NotFound_Creates(t *testing.T) {
	loginServer := "newregistry.azurecr.io"
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createFunc: func(_ context.Context, _, _ string, _ armcontainerregistry.Registry, _ *armcontainerregistry.RegistriesClientBeginCreateOptions) (*armcontainerregistry.RegistriesClientCreateResponse, error) {
			return &armcontainerregistry.RegistriesClientCreateResponse{
				Registry: armcontainerregistry.Registry{
					Properties: &armcontainerregistry.RegistryProperties{
						LoginServer: &loginServer,
					},
				},
			}, nil
		},
	}
	adapter := &ACRAdapter{client: stub}
	uri, err := adapter.EnsureRepository(context.Background(), "rg", "newregistry", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "newregistry.azurecr.io/myrepo"
	if uri != want {
		t.Fatalf("got URI %q, want %q", uri, want)
	}
}

func TestACRAdapter_EnsureRepository_GetError(t *testing.T) {
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{}, errors.New("access denied")
		},
	}
	adapter := &ACRAdapter{client: stub}
	_, err := adapter.EnsureRepository(context.Background(), "rg", "myregistry", "myrepo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestACRAdapter_EnsureRepository_CreateError(t *testing.T) {
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
		createFunc: func(_ context.Context, _, _ string, _ armcontainerregistry.Registry, _ *armcontainerregistry.RegistriesClientBeginCreateOptions) (*armcontainerregistry.RegistriesClientCreateResponse, error) {
			return nil, errors.New("quota exceeded")
		},
	}
	adapter := &ACRAdapter{client: stub}
	_, err := adapter.EnsureRepository(context.Background(), "rg", "myregistry", "myrepo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestACRAdapter_GetLoginServer_Exists(t *testing.T) {
	loginServer := "myregistry.azurecr.io"
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{
				Registry: armcontainerregistry.Registry{
					Properties: &armcontainerregistry.RegistryProperties{
						LoginServer: &loginServer,
					},
				},
			}, nil
		},
	}
	adapter := &ACRAdapter{client: stub}
	server, err := adapter.GetLoginServer(context.Background(), "rg", "myregistry")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server != loginServer {
		t.Fatalf("got %q, want %q", server, loginServer)
	}
}

func TestACRAdapter_GetLoginServer_NotFound(t *testing.T) {
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
	}
	adapter := &ACRAdapter{client: stub}
	_, err := adapter.GetLoginServer(context.Background(), "rg", "myregistry")
	if !errors.Is(err, ErrRepositoryNotFound) {
		t.Fatalf("expected ErrRepositoryNotFound, got %v", err)
	}
}

func TestACRAdapter_GetLoginServer_Error(t *testing.T) {
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{}, errors.New("network error")
		},
	}
	adapter := &ACRAdapter{client: stub}
	_, err := adapter.GetLoginServer(context.Background(), "rg", "myregistry")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestACRAdapter_GetLoginServer_NilProperties(t *testing.T) {
	stub := &stubACRAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armcontainerregistry.RegistriesClientGetOptions) (armcontainerregistry.RegistriesClientGetResponse, error) {
			return armcontainerregistry.RegistriesClientGetResponse{
				Registry: armcontainerregistry.Registry{
					Properties: nil,
				},
			}, nil
		},
	}
	adapter := &ACRAdapter{client: stub}
	_, err := adapter.GetLoginServer(context.Background(), "rg", "myregistry")
	if err == nil {
		t.Fatal("expected error for nil properties, got nil")
	}
}

func TestStrPtr(t *testing.T) {
	s := "hello"
	p := strPtr(s)
	if *p != s {
		t.Fatalf("got %q, want %q", *p, s)
	}
}
