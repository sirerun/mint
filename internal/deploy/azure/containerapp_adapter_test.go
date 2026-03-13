package azure

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"
)

type stubContainerAppAPI struct {
	getFunc            func(ctx context.Context, rg, name string, opts *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error)
	createOrUpdateFunc func(ctx context.Context, rg, name string, envelope armappcontainers.ContainerApp, opts *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error)
}

func (s *stubContainerAppAPI) Get(ctx context.Context, rg, name string, opts *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error) {
	return s.getFunc(ctx, rg, name, opts)
}

func (s *stubContainerAppAPI) CreateOrUpdate(ctx context.Context, rg, name string, envelope armappcontainers.ContainerApp, opts *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
	return s.createOrUpdateFunc(ctx, rg, name, envelope, opts)
}

type stubRevisionPager struct {
	pages []armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse
	index int
}

func (s *stubRevisionPager) More() bool {
	return s.index < len(s.pages)
}

func (s *stubRevisionPager) NextPage(_ context.Context) (armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse, error) {
	if s.index >= len(s.pages) {
		return armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse{}, errors.New("no more pages")
	}
	page := s.pages[s.index]
	s.index++
	return page, nil
}

type stubRevisionsAPI struct {
	pager revisionPager
}

func (s *stubRevisionsAPI) NewListRevisionsPager(_, _ string, _ *armappcontainers.ContainerAppsRevisionsClientListRevisionsOptions) revisionPager {
	return s.pager
}

func TestContainerAppAdapter_InterfaceCompliance(t *testing.T) {
	var _ ContainerAppClient = (*ContainerAppAdapter)(nil)
}

func TestContainerAppAdapter_CreateOrUpdateApp(t *testing.T) {
	appID := "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.App/containerApps/myapp"
	appName := "myapp"
	fqdn := "myapp.azurecontainerapps.io"
	revision := "myapp--rev1"
	state := armappcontainers.ContainerAppProvisioningStateSucceeded

	stub := &stubContainerAppAPI{
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ContainerApp, _ *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
			return &armappcontainers.ContainerAppsClientCreateOrUpdateResponse{
				ContainerApp: armappcontainers.ContainerApp{
					ID:   &appID,
					Name: &appName,
					Properties: &armappcontainers.ContainerAppProperties{
						LatestRevisionFqdn: &fqdn,
						LatestRevisionName: &revision,
						ProvisioningState:  &state,
					},
				},
			}, nil
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	app, err := adapter.CreateOrUpdateApp(context.Background(), &CreateOrUpdateAppInput{
		ResourceGroup: "rg",
		AppName:       "myapp",
		Region:        "eastus",
		EnvironmentID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.App/managedEnvironments/myenv",
		ImageURI:      "myregistry.azurecr.io/myrepo:latest",
		Port:          8080,
		CPU:           "0.5",
		Memory:        "1Gi",
		MinInstances:  1,
		MaxInstances:  3,
		EnvVars:       map[string]string{"KEY": "value"},
		Args:          []string{"--transport", "sse"},
		Ingress:       &IngressConfig{External: true, TargetPort: 8080},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.ID != appID {
		t.Fatalf("got ID %q, want %q", app.ID, appID)
	}
	if app.FQDN != fqdn {
		t.Fatalf("got FQDN %q, want %q", app.FQDN, fqdn)
	}
	if app.LatestRevision != revision {
		t.Fatalf("got revision %q, want %q", app.LatestRevision, revision)
	}
}

func TestContainerAppAdapter_CreateOrUpdateApp_Error(t *testing.T) {
	stub := &stubContainerAppAPI{
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ContainerApp, _ *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
			return nil, errors.New("quota exceeded")
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	_, err := adapter.CreateOrUpdateApp(context.Background(), &CreateOrUpdateAppInput{
		ResourceGroup: "rg",
		AppName:       "myapp",
		Region:        "eastus",
		CPU:           "0.25",
		Memory:        "0.5Gi",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestContainerAppAdapter_CreateOrUpdateApp_NoIngress(t *testing.T) {
	appName := "myapp"
	stub := &stubContainerAppAPI{
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ContainerApp, _ *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
			return &armappcontainers.ContainerAppsClientCreateOrUpdateResponse{
				ContainerApp: armappcontainers.ContainerApp{
					Name:       &appName,
					Properties: &armappcontainers.ContainerAppProperties{},
				},
			}, nil
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	app, err := adapter.CreateOrUpdateApp(context.Background(), &CreateOrUpdateAppInput{
		ResourceGroup: "rg",
		AppName:       "myapp",
		Region:        "eastus",
		CPU:           "1",
		Memory:        "2Gi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Name != appName {
		t.Fatalf("got name %q, want %q", app.Name, appName)
	}
}

func TestContainerAppAdapter_GetApp_Exists(t *testing.T) {
	appName := "myapp"
	stub := &stubContainerAppAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error) {
			return armappcontainers.ContainerAppsClientGetResponse{
				ContainerApp: armappcontainers.ContainerApp{
					Name:       &appName,
					Properties: &armappcontainers.ContainerAppProperties{},
				},
			}, nil
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	app, err := adapter.GetApp(context.Background(), "rg", "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Name != appName {
		t.Fatalf("got name %q, want %q", app.Name, appName)
	}
}

func TestContainerAppAdapter_GetApp_NotFound(t *testing.T) {
	stub := &stubContainerAppAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error) {
			return armappcontainers.ContainerAppsClientGetResponse{}, &azcore.ResponseError{StatusCode: 404}
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	_, err := adapter.GetApp(context.Background(), "rg", "myapp")
	if !errors.Is(err, ErrAppNotFound) {
		t.Fatalf("expected ErrAppNotFound, got %v", err)
	}
}

func TestContainerAppAdapter_GetApp_Error(t *testing.T) {
	stub := &stubContainerAppAPI{
		getFunc: func(_ context.Context, _, _ string, _ *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error) {
			return armappcontainers.ContainerAppsClientGetResponse{}, errors.New("network error")
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	_, err := adapter.GetApp(context.Background(), "rg", "myapp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestContainerAppAdapter_ListRevisions(t *testing.T) {
	revName := "myapp--rev1"
	active := true
	weight := int32(100)
	pager := &stubRevisionPager{
		pages: []armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse{
			{
				RevisionCollection: armappcontainers.RevisionCollection{
					Value: []*armappcontainers.Revision{
						{
							Name: &revName,
							Properties: &armappcontainers.RevisionProperties{
								Active:        &active,
								TrafficWeight: &weight,
							},
						},
					},
				},
			},
		},
	}
	adapter := &ContainerAppAdapter{
		revisions: &stubRevisionsAPI{pager: pager},
	}
	revisions, err := adapter.ListRevisions(context.Background(), "rg", "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(revisions))
	}
	if revisions[0].Name != revName {
		t.Fatalf("got revision name %q, want %q", revisions[0].Name, revName)
	}
	if !revisions[0].Active {
		t.Fatal("expected revision to be active")
	}
	if revisions[0].TrafficWeight != 100 {
		t.Fatalf("got traffic weight %d, want 100", revisions[0].TrafficWeight)
	}
}

func TestContainerAppAdapter_ListRevisions_Error(t *testing.T) {
	pager := &stubRevisionPager{
		pages: []armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse{{}},
	}
	// Override to return error
	adapter := &ContainerAppAdapter{
		revisions: &errorRevisionsAPI{},
	}
	_ = pager // unused in this test path
	_, err := adapter.ListRevisions(context.Background(), "rg", "myapp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type errorRevisionPager struct{}

func (e *errorRevisionPager) More() bool { return true }
func (e *errorRevisionPager) NextPage(_ context.Context) (armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse, error) {
	return armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse{}, errors.New("list failed")
}

type errorRevisionsAPI struct{}

func (e *errorRevisionsAPI) NewListRevisionsPager(_, _ string, _ *armappcontainers.ContainerAppsRevisionsClientListRevisionsOptions) revisionPager {
	return &errorRevisionPager{}
}

func TestContainerAppAdapter_UpdateTrafficSplit(t *testing.T) {
	stub := &stubContainerAppAPI{
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ContainerApp, _ *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
			return &armappcontainers.ContainerAppsClientCreateOrUpdateResponse{}, nil
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	err := adapter.UpdateTrafficSplit(context.Background(), "rg", "myapp", []TrafficWeight{
		{RevisionName: "rev1", Weight: 80},
		{RevisionName: "rev2", Weight: 20, Latest: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContainerAppAdapter_UpdateTrafficSplit_Error(t *testing.T) {
	stub := &stubContainerAppAPI{
		createOrUpdateFunc: func(_ context.Context, _, _ string, _ armappcontainers.ContainerApp, _ *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
			return nil, errors.New("update failed")
		},
	}
	adapter := &ContainerAppAdapter{client: stub}
	err := adapter.UpdateTrafficSplit(context.Background(), "rg", "myapp", []TrafficWeight{
		{RevisionName: "rev1", Weight: 100},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseCPU(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0.25", 0.25},
		{"0.5", 0.5},
		{"1.0", 1.0},
		{"1", 1.0},
		{"2.0", 2.0},
		{"2", 2.0},
		{"4.0", 4.0},
		{"4", 4.0},
		{"unknown", 0.25},
		{"", 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCPU(tt.input)
			if got != tt.want {
				t.Fatalf("parseCPU(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestContainerAppFromSDK_NilFields(t *testing.T) {
	app := containerAppFromSDK(armappcontainers.ContainerApp{})
	if app.ID != "" || app.Name != "" || app.FQDN != "" || app.LatestRevision != "" || app.ProvisioningState != "" {
		t.Fatal("expected all empty fields for nil SDK app")
	}
}

func TestBuildEnvVars(t *testing.T) {
	vars := buildEnvVars(map[string]string{"KEY": "value"}, []SecretRef{{Name: "secret1", KeyVault: "vault", Identity: "id"}})
	if len(vars) != 2 {
		t.Fatalf("expected 2 env vars, got %d", len(vars))
	}
}

func TestHelperPtrs(t *testing.T) {
	f := float64Ptr(1.5)
	if *f != 1.5 {
		t.Fatalf("float64Ptr: got %v, want 1.5", *f)
	}
	i := int32Ptr(42)
	if *i != 42 {
		t.Fatalf("int32Ptr: got %v, want 42", *i)
	}
	b := boolPtr(true)
	if !*b {
		t.Fatal("boolPtr: got false, want true")
	}
}
