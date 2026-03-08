// Package gcp provides helpers for deploying to Google Cloud Platform.
package gcp

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// IAMClient handles service account operations.
type IAMClient interface {
	// GetServiceAccount returns the email of a service account by its resource name.
	// The name format is "projects/<project>/serviceAccounts/<email>".
	// Returns empty string and nil error if the account does not exist.
	GetServiceAccount(ctx context.Context, name string) (email string, err error)

	// CreateServiceAccount creates a new service account in the given project.
	// Returns the email of the newly created service account.
	CreateServiceAccount(ctx context.Context, projectID, accountID, displayName string) (email string, err error)
}

// WorkloadIdentityConfig holds the configuration for Workload Identity Federation.
type WorkloadIdentityConfig struct {
	ProjectID      string // GCP project ID
	ProjectNumber  string // GCP project number
	PoolID         string // default: "mint-github-pool"
	ProviderID     string // default: "mint-github-provider"
	GitHubOrg      string // e.g., "sirerun"
	GitHubRepo     string // e.g., "mint"
	ServiceAccount string // created SA email (output)
}

// WorkloadIdentityResult holds the output needed by GitHub Actions.
type WorkloadIdentityResult struct {
	ProviderName   string // full resource name of the provider
	ServiceAccount string // SA email for workload identity
}

const (
	defaultPoolID     = "mint-github-pool"
	defaultProviderID = "mint-github-provider"
	saAccountID       = "mint-deploy"
)

// EnsureWorkloadIdentity creates the service account needed for Workload Identity
// Federation and prints gcloud commands for completing the pool/provider setup.
// It is idempotent: safe to call multiple times.
func EnsureWorkloadIdentity(ctx context.Context, iamClient IAMClient, config WorkloadIdentityConfig, stderr io.Writer) (*WorkloadIdentityResult, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	if config.ProjectNumber == "" {
		return nil, fmt.Errorf("project number is required")
	}

	poolID := config.PoolID
	if poolID == "" {
		poolID = defaultPoolID
	}
	providerID := config.ProviderID
	if providerID == "" {
		providerID = defaultProviderID
	}

	saEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", saAccountID, config.ProjectID)
	resourceName := fmt.Sprintf("projects/%s/serviceAccounts/%s", config.ProjectID, saEmail)

	// Check if the service account already exists.
	email, err := iamClient.GetServiceAccount(ctx, resourceName)
	if err != nil {
		return nil, fmt.Errorf("checking service account: %w", err)
	}

	if email == "" {
		// Create the service account.
		email, err = iamClient.CreateServiceAccount(ctx, config.ProjectID, saAccountID, "Mint deploy service account")
		if err != nil {
			return nil, fmt.Errorf("creating service account: %w", err)
		}
	}

	providerName := fmt.Sprintf(
		"projects/%s/locations/global/workloadIdentityPools/%s/providers/%s",
		config.ProjectNumber, poolID, providerID,
	)

	result := &WorkloadIdentityResult{
		ProviderName:   providerName,
		ServiceAccount: email,
	}

	// Print gcloud commands for completing Workload Identity Federation setup.
	printSetupInstructions(stderr, config.ProjectID, poolID, providerID, email, config.GitHubOrg, config.GitHubRepo)

	return result, nil
}

func printSetupInstructions(w io.Writer, projectID, poolID, providerID, saEmail, githubOrg, githubRepo string) {
	var b strings.Builder
	b.WriteString("\n--- Workload Identity Federation Setup ---\n")
	b.WriteString("Run the following gcloud commands to complete setup:\n\n")

	fmt.Fprintf(&b, "gcloud iam workload-identity-pools create %s \\\n  --location=global --project=%s\n\n", poolID, projectID)

	fmt.Fprintf(&b, "gcloud iam workload-identity-pools providers create-oidc %s \\\n", providerID)
	fmt.Fprintf(&b, "  --location=global \\\n")
	fmt.Fprintf(&b, "  --workload-identity-pool=%s \\\n", poolID)
	fmt.Fprintf(&b, "  --issuer-uri=\"https://token.actions.githubusercontent.com\" \\\n")
	fmt.Fprintf(&b, "  --attribute-mapping=\"google.subject=assertion.sub,attribute.repository=assertion.repository\" \\\n")
	fmt.Fprintf(&b, "  --project=%s\n\n", projectID)

	if githubOrg != "" && githubRepo != "" {
		fmt.Fprintf(&b, "gcloud iam service-accounts add-iam-policy-binding %s \\\n", saEmail)
		fmt.Fprintf(&b, "  --role=roles/iam.workloadIdentityUser \\\n")
		fmt.Fprintf(&b, "  --member=\"principalSet://iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/attribute.repository/%s/%s\" \\\n",
			projectID, poolID, githubOrg, githubRepo)
		fmt.Fprintf(&b, "  --project=%s\n", projectID)
	}

	b.WriteString("\n--- End Setup Instructions ---\n")
	_, _ = fmt.Fprint(w, b.String())
}
