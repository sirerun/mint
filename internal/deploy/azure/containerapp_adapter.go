package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers"
)

// containerAppAPI abstracts the Azure Container Apps SDK methods used by ContainerAppAdapter.
type containerAppAPI interface {
	Get(ctx context.Context, resourceGroupName string, containerAppName string, options *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error)
	CreateOrUpdate(ctx context.Context, resourceGroupName string, containerAppName string, containerAppEnvelope armappcontainers.ContainerApp, options *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error)
}

// containerAppRevisionsAPI abstracts the revisions SDK methods.
type containerAppRevisionsAPI interface {
	NewListRevisionsPager(resourceGroupName string, containerAppName string, options *armappcontainers.ContainerAppsRevisionsClientListRevisionsOptions) revisionPager
}

// revisionPager abstracts paging over revisions.
type revisionPager interface {
	More() bool
	NextPage(ctx context.Context) (armappcontainers.ContainerAppsRevisionsClientListRevisionsResponse, error)
}

// containerAppSDKAdapter wraps the SDK poller-based CreateOrUpdate.
type containerAppSDKAdapter struct {
	client *armappcontainers.ContainerAppsClient
}

func (a *containerAppSDKAdapter) Get(ctx context.Context, resourceGroupName, containerAppName string, options *armappcontainers.ContainerAppsClientGetOptions) (armappcontainers.ContainerAppsClientGetResponse, error) {
	return a.client.Get(ctx, resourceGroupName, containerAppName, options)
}

func (a *containerAppSDKAdapter) CreateOrUpdate(ctx context.Context, resourceGroupName, containerAppName string, envelope armappcontainers.ContainerApp, options *armappcontainers.ContainerAppsClientBeginCreateOrUpdateOptions) (*armappcontainers.ContainerAppsClientCreateOrUpdateResponse, error) {
	poller, err := a.client.BeginCreateOrUpdate(ctx, resourceGroupName, containerAppName, envelope, options)
	if err != nil {
		return nil, err
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// revisionsSDKAdapter wraps the real revisions client.
type revisionsSDKAdapter struct {
	client *armappcontainers.ContainerAppsRevisionsClient
}

func (a *revisionsSDKAdapter) NewListRevisionsPager(resourceGroupName, containerAppName string, options *armappcontainers.ContainerAppsRevisionsClientListRevisionsOptions) revisionPager {
	return a.client.NewListRevisionsPager(resourceGroupName, containerAppName, options)
}

// ContainerAppAdapter implements ContainerAppClient using the Azure SDK.
type ContainerAppAdapter struct {
	client    containerAppAPI
	revisions containerAppRevisionsAPI
}

var _ ContainerAppClient = (*ContainerAppAdapter)(nil)

// NewContainerAppAdapter creates a new Container App adapter backed by the Azure SDK.
func NewContainerAppAdapter(subscriptionID string, cred azcore.TokenCredential) (*ContainerAppAdapter, error) {
	client, err := armappcontainers.NewContainerAppsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create container apps client: %w", err)
	}
	revClient, err := armappcontainers.NewContainerAppsRevisionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("create revisions client: %w", err)
	}
	return &ContainerAppAdapter{
		client:    &containerAppSDKAdapter{client: client},
		revisions: &revisionsSDKAdapter{client: revClient},
	}, nil
}

// CreateOrUpdateApp creates or updates a Container App.
func (a *ContainerAppAdapter) CreateOrUpdateApp(ctx context.Context, input *CreateOrUpdateAppInput) (*ContainerApp, error) {
	containers := []*armappcontainers.Container{
		{
			Name:  strPtr(input.AppName),
			Image: strPtr(input.ImageURI),
			Resources: &armappcontainers.ContainerResources{
				CPU:    float64Ptr(parseCPU(input.CPU)),
				Memory: strPtr(input.Memory),
			},
			Env: buildEnvVars(input.EnvVars, input.SecretRefs),
		},
	}
	if len(input.Args) > 0 {
		argPtrs := make([]*string, len(input.Args))
		for i, arg := range input.Args {
			argPtrs[i] = strPtr(arg)
		}
		containers[0].Args = argPtrs
	}

	template := &armappcontainers.Template{
		Containers: containers,
		Scale: &armappcontainers.Scale{
			MinReplicas: int32Ptr(int32(input.MinInstances)),
			MaxReplicas: int32Ptr(int32(input.MaxInstances)),
		},
	}

	config := &armappcontainers.Configuration{}
	if input.Ingress != nil {
		config.Ingress = &armappcontainers.Ingress{
			External:   boolPtr(input.Ingress.External),
			TargetPort: int32Ptr(int32(input.Ingress.TargetPort)),
		}
	}

	envelope := armappcontainers.ContainerApp{
		Location: strPtr(input.Region),
		Properties: &armappcontainers.ContainerAppProperties{
			ManagedEnvironmentID: strPtr(input.EnvironmentID),
			Configuration:        config,
			Template:             template,
		},
	}

	resp, err := a.client.CreateOrUpdate(ctx, input.ResourceGroup, input.AppName, envelope, nil)
	if err != nil {
		return nil, fmt.Errorf("create or update app: %w", err)
	}
	return containerAppFromSDK(resp.ContainerApp), nil
}

// GetApp returns metadata for a Container App.
func (a *ContainerAppAdapter) GetApp(ctx context.Context, resourceGroup, appName string) (*ContainerApp, error) {
	resp, err := a.client.Get(ctx, resourceGroup, appName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == 404 {
			return nil, ErrAppNotFound
		}
		return nil, fmt.Errorf("get app: %w", err)
	}
	return containerAppFromSDK(resp.ContainerApp), nil
}

// ListRevisions returns all revisions for a Container App.
func (a *ContainerAppAdapter) ListRevisions(ctx context.Context, resourceGroup, appName string) ([]Revision, error) {
	pager := a.revisions.NewListRevisionsPager(resourceGroup, appName, nil)
	var revisions []Revision
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list revisions: %w", err)
		}
		for _, r := range page.Value {
			rev := Revision{}
			if r.Name != nil {
				rev.Name = *r.Name
			}
			if r.Properties != nil {
				rev.Active = r.Properties.Active != nil && *r.Properties.Active
				if r.Properties.TrafficWeight != nil {
					rev.TrafficWeight = int(*r.Properties.TrafficWeight)
				}
				if r.Properties.CreatedTime != nil {
					rev.CreatedTime = r.Properties.CreatedTime.String()
				}
			}
			revisions = append(revisions, rev)
		}
	}
	return revisions, nil
}

// UpdateTrafficSplit updates the traffic distribution across revisions.
func (a *ContainerAppAdapter) UpdateTrafficSplit(ctx context.Context, resourceGroup, appName string, traffic []TrafficWeight) error {
	sdkTraffic := make([]*armappcontainers.TrafficWeight, len(traffic))
	for i, tw := range traffic {
		sdkTraffic[i] = &armappcontainers.TrafficWeight{
			RevisionName:   strPtr(tw.RevisionName),
			Weight:         int32Ptr(int32(tw.Weight)),
			LatestRevision: boolPtr(tw.Latest),
		}
	}

	_, err := a.client.CreateOrUpdate(ctx, resourceGroup, appName, armappcontainers.ContainerApp{
		Properties: &armappcontainers.ContainerAppProperties{
			Configuration: &armappcontainers.Configuration{
				Ingress: &armappcontainers.Ingress{
					Traffic: sdkTraffic,
				},
			},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("update traffic: %w", err)
	}
	return nil
}

func containerAppFromSDK(app armappcontainers.ContainerApp) *ContainerApp {
	result := &ContainerApp{}
	if app.ID != nil {
		result.ID = *app.ID
	}
	if app.Name != nil {
		result.Name = *app.Name
	}
	if app.Properties != nil {
		if app.Properties.LatestRevisionFqdn != nil {
			result.FQDN = *app.Properties.LatestRevisionFqdn
		}
		if app.Properties.LatestRevisionName != nil {
			result.LatestRevision = *app.Properties.LatestRevisionName
		}
		if app.Properties.ProvisioningState != nil {
			result.ProvisioningState = string(*app.Properties.ProvisioningState)
		}
	}
	return result
}

func buildEnvVars(envVars map[string]string, secretRefs []SecretRef) []*armappcontainers.EnvironmentVar {
	vars := make([]*armappcontainers.EnvironmentVar, 0, len(envVars)+len(secretRefs))
	for k, v := range envVars {
		vars = append(vars, &armappcontainers.EnvironmentVar{
			Name:  strPtr(k),
			Value: strPtr(v),
		})
	}
	for _, ref := range secretRefs {
		vars = append(vars, &armappcontainers.EnvironmentVar{
			Name:      strPtr(ref.Name),
			SecretRef: strPtr(ref.Name),
		})
	}
	return vars
}

func parseCPU(cpu string) float64 {
	switch cpu {
	case "0.25":
		return 0.25
	case "0.5":
		return 0.5
	case "1.0", "1":
		return 1.0
	case "2.0", "2":
		return 2.0
	case "4.0", "4":
		return 4.0
	default:
		return 0.25
	}
}

func float64Ptr(f float64) *float64 { return &f }
func int32Ptr(i int32) *int32       { return &i }
func boolPtr(b bool) *bool          { return &b }
