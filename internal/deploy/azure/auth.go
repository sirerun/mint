package azure

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// newDefaultCredential is a package-level variable for testing the credential creation path.
var newDefaultCredential = func(options *azidentity.DefaultAzureCredentialOptions) (azcore.TokenCredential, error) {
	return azidentity.NewDefaultAzureCredential(options)
}

// getSubscriptionID is a package-level variable for testing environment variable lookups.
var getSubscriptionID = func() string {
	return os.Getenv("AZURE_SUBSCRIPTION_ID")
}

// getResourceGroup is a package-level variable for testing environment variable lookups.
var getResourceGroup = func() string {
	return os.Getenv("AZURE_RESOURCE_GROUP")
}

// Credentials wraps resolved Azure authentication details.
type Credentials struct {
	SubscriptionID string
	ResourceGroup  string
	TenantID       string
	Credential     azcore.TokenCredential
}

// Authenticate resolves Azure credentials via the SDK default chain:
// environment variables, Azure CLI, and managed identity.
func Authenticate(ctx context.Context, stderr io.Writer) (*Credentials, error) {
	subscriptionID := getSubscriptionID()
	if subscriptionID == "" {
		return nil, fmt.Errorf("AZURE_SUBSCRIPTION_ID is required. Set it via environment variable")
	}

	resourceGroup := getResourceGroup()
	if resourceGroup == "" {
		return nil, fmt.Errorf("AZURE_RESOURCE_GROUP is required. Set it via environment variable")
	}

	cred, err := newDefaultCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("azure credentials not found. Configure credentials via environment variables, Azure CLI, or managed identity: %w", err)
	}

	tenantID := os.Getenv("AZURE_TENANT_ID")

	_, _ = fmt.Fprintf(stderr, "Authenticated with Azure (subscription: %s, resource group: %s)\n", subscriptionID, resourceGroup)

	return &Credentials{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		TenantID:       tenantID,
		Credential:     cred,
	}, nil
}
