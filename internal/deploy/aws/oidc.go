package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrOIDCProviderNotFound indicates that an OIDC provider was not found.
var ErrOIDCProviderNotFound = errors.New("oidc: provider not found")

// oidcJSONMarshal is a package-level variable for testing the JSON marshal error path.
var oidcJSONMarshal = json.Marshal

// OIDCClient abstracts AWS IAM OIDC provider operations.
type OIDCClient interface {
	// GetOpenIDConnectProvider checks if an OIDC provider exists by ARN.
	// Returns ErrOIDCProviderNotFound if it does not exist.
	GetOpenIDConnectProvider(ctx context.Context, arn string) error

	// CreateOpenIDConnectProvider creates an OIDC identity provider.
	// Returns the ARN of the created provider.
	CreateOpenIDConnectProvider(ctx context.Context, url string, thumbprints []string) (string, error)

	// CreateRole creates a new IAM role.
	CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error)

	// GetRole returns an existing IAM role.
	// Returns ErrRoleNotFound if the role does not exist.
	GetRole(ctx context.Context, roleName string) (*Role, error)

	// AttachRolePolicy attaches a managed policy to a role.
	AttachRolePolicy(ctx context.Context, roleName, policyARN string) error
}

// OIDCConfig holds configuration for setting up GitHub Actions OIDC.
type OIDCConfig struct {
	AccountID string
	Region    string
	RepoOwner string // GitHub org/user
	RepoName  string // GitHub repo name
}

// OIDCResult holds the output of OIDC provider setup.
type OIDCResult struct {
	ProviderARN string
	RoleARN     string
	RoleName    string
}

const (
	githubOIDCURL        = "https://token.actions.githubusercontent.com"
	githubOIDCThumbprint = "6938fd4d98bab03faadb97b34396831e3780aea1"
	oidcRolePrefix       = "mint-github-deploy-"
)

// EnsureOIDCProvider sets up GitHub Actions OIDC federation for AWS.
// It creates the OIDC identity provider (if needed), an IAM role with
// a trust policy scoped to the specified repository, and attaches
// required policies for ECS Fargate deployment.
func EnsureOIDCProvider(ctx context.Context, client OIDCClient, config OIDCConfig, stderr io.Writer) (*OIDCResult, error) {
	if err := validateOIDCConfig(config); err != nil {
		return nil, err
	}

	providerARN := fmt.Sprintf(
		"arn:aws:iam::%s:oidc-provider/token.actions.githubusercontent.com",
		config.AccountID,
	)

	// Check if OIDC provider exists; create if not.
	err := client.GetOpenIDConnectProvider(ctx, providerARN)
	if err != nil {
		if !errors.Is(err, ErrOIDCProviderNotFound) {
			return nil, fmt.Errorf("checking OIDC provider: %w", err)
		}

		createdARN, createErr := client.CreateOpenIDConnectProvider(
			ctx,
			githubOIDCURL,
			[]string{githubOIDCThumbprint},
		)
		if createErr != nil {
			return nil, fmt.Errorf("creating OIDC provider: %w", createErr)
		}
		providerARN = createdARN

		_, _ = fmt.Fprintf(stderr, "Created OIDC provider: %s\n", providerARN)
	}

	// Ensure IAM role with trust policy for the repo.
	roleName := oidcRolePrefix + config.RepoName
	trustPolicy, err := githubOIDCTrustPolicy(providerARN, config.RepoOwner, config.RepoName)
	if err != nil {
		return nil, fmt.Errorf("building trust policy: %w", err)
	}

	role, err := ensureOIDCRole(ctx, client, &CreateRoleInput{
		RoleName:                 roleName,
		AssumeRolePolicyDocument: trustPolicy,
		Description:              fmt.Sprintf("Mint GitHub Actions deploy role for %s/%s", config.RepoOwner, config.RepoName),
	}, stderr)
	if err != nil {
		return nil, fmt.Errorf("ensuring deploy role: %w", err)
	}

	// Attach required policies for ECS deployment.
	policies := []string{
		"arn:aws:iam::aws:policy/AmazonECS_FullAccess",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser",
	}
	for _, policyARN := range policies {
		if attachErr := client.AttachRolePolicy(ctx, roleName, policyARN); attachErr != nil {
			return nil, fmt.Errorf("attaching policy %s: %w", policyARN, attachErr)
		}
	}

	return &OIDCResult{
		ProviderARN: providerARN,
		RoleARN:     role.ARN,
		RoleName:    role.RoleName,
	}, nil
}

func validateOIDCConfig(c OIDCConfig) error {
	switch {
	case c.AccountID == "":
		return fmt.Errorf("accountID is required")
	case c.Region == "":
		return fmt.Errorf("region is required")
	case c.RepoOwner == "":
		return fmt.Errorf("repoOwner is required")
	case c.RepoName == "":
		return fmt.Errorf("repoName is required")
	}
	return nil
}

func ensureOIDCRole(ctx context.Context, client OIDCClient, input *CreateRoleInput, stderr io.Writer) (*Role, error) {
	role, err := client.GetRole(ctx, input.RoleName)
	if err == nil {
		return role, nil
	}
	if !errors.Is(err, ErrRoleNotFound) {
		return nil, err
	}

	role, err = client.CreateRole(ctx, input)
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(stderr, "Created IAM role: %s\n", role.RoleName)
	return role, nil
}

func githubOIDCTrustPolicy(providerARN, repoOwner, repoName string) (string, error) {
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Effect": "Allow",
				"Principal": map[string]string{
					"Federated": providerARN,
				},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": map[string]map[string]string{
					"StringEquals": {
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
					},
					"StringLike": {
						"token.actions.githubusercontent.com:sub": fmt.Sprintf(
							"repo:%s/%s:*",
							repoOwner, repoName,
						),
					},
				},
			},
		},
	}
	b, err := oidcJSONMarshal(policy)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PrintOIDCInstructions outputs manual setup instructions for the user.
func PrintOIDCInstructions(w io.Writer, result *OIDCResult) {
	var b strings.Builder
	b.WriteString("\n--- AWS OIDC Setup Complete ---\n")
	fmt.Fprintf(&b, "OIDC Provider ARN: %s\n", result.ProviderARN)
	fmt.Fprintf(&b, "IAM Role ARN:      %s\n", result.RoleARN)
	fmt.Fprintf(&b, "IAM Role Name:     %s\n", result.RoleName)
	b.WriteString("\nUse the Role ARN in your GitHub Actions workflow:\n")
	fmt.Fprintf(&b, "  role-to-assume: '%s'\n", result.RoleARN)
	b.WriteString("--- End Setup ---\n")
	_, _ = fmt.Fprint(w, b.String())
}
