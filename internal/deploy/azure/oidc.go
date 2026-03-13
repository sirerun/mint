package azure

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// OIDCClient abstracts Azure AD app registration and federated credential operations.
type OIDCClient interface {
	// GetAppRegistration checks if an Azure AD app registration exists by display name.
	// Returns ErrADAppNotFound if it does not exist.
	GetAppRegistration(ctx context.Context, displayName string) (*AppRegistration, error)

	// CreateAppRegistration creates a new Azure AD app registration.
	CreateAppRegistration(ctx context.Context, displayName string) (*AppRegistration, error)

	// CreateFederatedCredential creates a federated identity credential on the app.
	CreateFederatedCredential(ctx context.Context, appID string, input *FederatedCredentialInput) error

	// AssignRole assigns an Azure RBAC role to a service principal.
	AssignRole(ctx context.Context, input *RoleAssignmentInput) error
}

// ErrADAppNotFound indicates that an Azure AD app registration was not found.
var ErrADAppNotFound = fmt.Errorf("azure ad: app registration not found")

// AppRegistration represents an Azure AD app registration.
type AppRegistration struct {
	AppID    string // Application (client) ID
	ObjectID string // Object ID
}

// FederatedCredentialInput holds parameters for creating a federated identity credential.
type FederatedCredentialInput struct {
	Name      string
	Issuer    string
	Subject   string
	Audiences []string
}

// RoleAssignmentInput holds parameters for assigning an Azure RBAC role.
type RoleAssignmentInput struct {
	PrincipalID    string
	RoleName       string
	Scope          string
	SubscriptionID string
	ResourceGroup  string
}

// OIDCConfig holds configuration for setting up GitHub Actions OIDC with Azure.
type OIDCConfig struct {
	SubscriptionID string
	TenantID       string
	RepoOwner      string // GitHub org/user
	RepoName       string // GitHub repo name
}

// OIDCResult holds the output of OIDC provider setup.
type OIDCResult struct {
	ClientID       string
	TenantID       string
	SubscriptionID string
}

const (
	githubOIDCIssuer = "https://token.actions.githubusercontent.com"
	oidcAppPrefix    = "mint-github-deploy-"
)

// EnsureOIDCProvider sets up GitHub Actions OIDC federation for Azure.
// It creates an Azure AD app registration (if needed), adds a federated
// credential for the specified GitHub repository, and assigns the
// Contributor role on the subscription scope.
func EnsureOIDCProvider(ctx context.Context, client OIDCClient, config OIDCConfig, stderr io.Writer) (*OIDCResult, error) {
	if err := validateOIDCConfig(config); err != nil {
		return nil, err
	}

	displayName := oidcAppPrefix + config.RepoName

	// Check if app registration exists; create if not.
	app, err := client.GetAppRegistration(ctx, displayName)
	if err != nil {
		if err != ErrADAppNotFound {
			return nil, fmt.Errorf("checking app registration: %w", err)
		}

		app, err = client.CreateAppRegistration(ctx, displayName)
		if err != nil {
			return nil, fmt.Errorf("creating app registration: %w", err)
		}
		_, _ = fmt.Fprintf(stderr, "Created Azure AD app registration: %s (client ID: %s)\n", displayName, app.AppID)
	}

	// Create federated credential for GitHub Actions OIDC.
	credName := fmt.Sprintf("github-%s-%s", config.RepoOwner, config.RepoName)
	subject := fmt.Sprintf("repo:%s/%s:ref:refs/heads/main", config.RepoOwner, config.RepoName)
	err = client.CreateFederatedCredential(ctx, app.AppID, &FederatedCredentialInput{
		Name:      credName,
		Issuer:    githubOIDCIssuer,
		Subject:   subject,
		Audiences: []string{"api://AzureADTokenExchange"},
	})
	if err != nil {
		return nil, fmt.Errorf("creating federated credential: %w", err)
	}
	_, _ = fmt.Fprintf(stderr, "Configured federated credential: %s\n", credName)

	// Assign Contributor role.
	scope := fmt.Sprintf("/subscriptions/%s", config.SubscriptionID)
	err = client.AssignRole(ctx, &RoleAssignmentInput{
		PrincipalID:    app.ObjectID,
		RoleName:       "Contributor",
		Scope:          scope,
		SubscriptionID: config.SubscriptionID,
	})
	if err != nil {
		return nil, fmt.Errorf("assigning Contributor role: %w", err)
	}
	_, _ = fmt.Fprintf(stderr, "Assigned Contributor role on subscription %s\n", config.SubscriptionID)

	return &OIDCResult{
		ClientID:       app.AppID,
		TenantID:       config.TenantID,
		SubscriptionID: config.SubscriptionID,
	}, nil
}

// PrintOIDCInstructions outputs setup instructions for the user.
func PrintOIDCInstructions(w io.Writer, result *OIDCResult) {
	var b strings.Builder
	b.WriteString("\n--- Azure OIDC Setup Complete ---\n")
	fmt.Fprintf(&b, "Client ID:       %s\n", result.ClientID)
	fmt.Fprintf(&b, "Tenant ID:       %s\n", result.TenantID)
	fmt.Fprintf(&b, "Subscription ID: %s\n", result.SubscriptionID)
	b.WriteString("\nAdd these as GitHub Actions secrets:\n")
	fmt.Fprintf(&b, "  AZURE_CLIENT_ID:       %s\n", result.ClientID)
	fmt.Fprintf(&b, "  AZURE_TENANT_ID:       %s\n", result.TenantID)
	fmt.Fprintf(&b, "  AZURE_SUBSCRIPTION_ID: %s\n", result.SubscriptionID)
	b.WriteString("--- End Setup ---\n")
	_, _ = fmt.Fprint(w, b.String())
}

func validateOIDCConfig(c OIDCConfig) error {
	switch {
	case c.SubscriptionID == "":
		return fmt.Errorf("subscriptionID is required")
	case c.TenantID == "":
		return fmt.Errorf("tenantID is required")
	case c.RepoOwner == "":
		return fmt.Errorf("repoOwner is required")
	case c.RepoName == "":
		return fmt.Errorf("repoName is required")
	}
	return nil
}
