package gcp

import (
	"context"
	"fmt"
	"os"
)

// IAMPolicyClient abstracts IAM policy operations for Cloud Run services.
type IAMPolicyClient interface {
	GetIAMPolicy(ctx context.Context, serviceName string) (*IAMPolicy, error)
	SetIAMPolicy(ctx context.Context, serviceName string, policy *IAMPolicy) error
}

// IAMPolicy represents a simplified IAM policy.
type IAMPolicy struct {
	Bindings []IAMBinding
}

// IAMBinding is a role-members pair.
type IAMBinding struct {
	Role    string
	Members []string
}

// ServiceAccountConfig holds config for creating a service account.
type ServiceAccountConfig struct {
	ProjectID   string
	ServiceName string // used to derive SA name: mint-mcp-<serviceName>
	Public      bool   // if true, add allUsers as invoker
}

const (
	roleRunInvoker = "roles/run.invoker"
	memberAllUsers = "allUsers"
	memberAllAuth  = "allAuthenticatedUsers"
)

// ConfigureIAMPolicy configures the IAM policy for a Cloud Run service.
// If Public is true, it adds allUsers as an invoker and prints a warning.
// If Public is false, it removes allUsers and allAuthenticatedUsers from the invoker role.
func ConfigureIAMPolicy(ctx context.Context, client IAMPolicyClient, config ServiceAccountConfig, cloudRunServiceName string) error {
	policy, err := client.GetIAMPolicy(ctx, cloudRunServiceName)
	if err != nil {
		return fmt.Errorf("get IAM policy: %w", err)
	}

	if config.Public {
		addPublicInvoker(policy)
		fmt.Fprintln(os.Stderr, "WARNING: Service is publicly accessible without authentication")
	} else {
		removePublicInvokers(policy)
	}

	if err := client.SetIAMPolicy(ctx, cloudRunServiceName, policy); err != nil {
		return fmt.Errorf("set IAM policy: %w", err)
	}

	return nil
}

// ServiceAccountEmail returns the email for a mint-managed service account.
func ServiceAccountEmail(projectID, serviceName string) string {
	return fmt.Sprintf("mint-mcp-%s@%s.iam.gserviceaccount.com", serviceName, projectID)
}

// addPublicInvoker ensures allUsers is present in the run.invoker binding.
func addPublicInvoker(policy *IAMPolicy) {
	for i, b := range policy.Bindings {
		if b.Role == roleRunInvoker {
			for _, m := range b.Members {
				if m == memberAllUsers {
					return // already present
				}
			}
			policy.Bindings[i].Members = append(policy.Bindings[i].Members, memberAllUsers)
			return
		}
	}
	// No existing binding for the role; create one.
	policy.Bindings = append(policy.Bindings, IAMBinding{
		Role:    roleRunInvoker,
		Members: []string{memberAllUsers},
	})
}

// removePublicInvokers removes allUsers and allAuthenticatedUsers from the
// run.invoker binding.
func removePublicInvokers(policy *IAMPolicy) {
	for i, b := range policy.Bindings {
		if b.Role != roleRunInvoker {
			continue
		}
		filtered := make([]string, 0, len(b.Members))
		for _, m := range b.Members {
			if m != memberAllUsers && m != memberAllAuth {
				filtered = append(filtered, m)
			}
		}
		policy.Bindings[i].Members = filtered
		return
	}
}
