package gcp

import (
	"context"
	"fmt"

	iam "cloud.google.com/go/iam/admin/apiv1"
	adminpb "cloud.google.com/go/iam/admin/apiv1/adminpb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	run "cloud.google.com/go/run/apiv2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IAMPolicyAdapter implements IAMPolicyClient using the Cloud Run Services API.
type IAMPolicyAdapter struct {
	services *run.ServicesClient
}

// Compile-time interface checks.
var _ IAMPolicyClient = (*IAMPolicyAdapter)(nil)
var _ IAMClient = (*IAMServiceAccountAdapter)(nil)

// NewIAMPolicyAdapter creates an IAMPolicyAdapter. It can share the same
// ServicesClient used by CloudRunServiceAdapter if desired, or create its own.
func NewIAMPolicyAdapter(services *run.ServicesClient) *IAMPolicyAdapter {
	return &IAMPolicyAdapter{services: services}
}

// GetIAMPolicy retrieves the IAM policy for a Cloud Run service.
func (a *IAMPolicyAdapter) GetIAMPolicy(ctx context.Context, serviceName string) (*IAMPolicy, error) {
	policy, err := a.services.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
		Resource: serviceName,
	})
	if err != nil {
		return nil, err
	}
	return iamPolicyFromPb(policy), nil
}

// SetIAMPolicy sets the IAM policy for a Cloud Run service.
func (a *IAMPolicyAdapter) SetIAMPolicy(ctx context.Context, serviceName string, policy *IAMPolicy) error {
	_, err := a.services.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
		Resource: serviceName,
		Policy:   iamPolicyToPb(policy),
	})
	return err
}

// IAMServiceAccountAdapter implements IAMClient using the IAM Admin API.
type IAMServiceAccountAdapter struct {
	client *iam.IamClient
}

// NewIAMServiceAccountAdapter creates an adapter for service account operations.
func NewIAMServiceAccountAdapter(ctx context.Context) (*IAMServiceAccountAdapter, error) {
	client, err := iam.NewIamClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating IAM client: %w", err)
	}
	return &IAMServiceAccountAdapter{client: client}, nil
}

// Close releases the underlying gRPC connection.
func (a *IAMServiceAccountAdapter) Close() error {
	return a.client.Close()
}

// GetServiceAccount returns the email of a service account. If the account does
// not exist, it returns an empty string and nil error.
func (a *IAMServiceAccountAdapter) GetServiceAccount(ctx context.Context, name string) (string, error) {
	sa, err := a.client.GetServiceAccount(ctx, &adminpb.GetServiceAccountRequest{
		Name: name,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", err
	}
	return sa.Email, nil
}

// CreateServiceAccount creates a new service account and returns its email.
func (a *IAMServiceAccountAdapter) CreateServiceAccount(ctx context.Context, projectID, accountID, displayName string) (string, error) {
	sa, err := a.client.CreateServiceAccount(ctx, &adminpb.CreateServiceAccountRequest{
		Name:      fmt.Sprintf("projects/%s", projectID),
		AccountId: accountID,
		ServiceAccount: &adminpb.ServiceAccount{
			DisplayName: displayName,
		},
	})
	if err != nil {
		return "", err
	}
	return sa.Email, nil
}

func iamPolicyFromPb(pb *iampb.Policy) *IAMPolicy {
	policy := &IAMPolicy{
		Bindings: make([]IAMBinding, len(pb.Bindings)),
	}
	for i, b := range pb.Bindings {
		members := make([]string, len(b.Members))
		copy(members, b.Members)
		policy.Bindings[i] = IAMBinding{
			Role:    b.Role,
			Members: members,
		}
	}
	return policy
}

func iamPolicyToPb(policy *IAMPolicy) *iampb.Policy {
	pb := &iampb.Policy{
		Bindings: make([]*iampb.Binding, len(policy.Bindings)),
	}
	for i, b := range policy.Bindings {
		members := make([]string, len(b.Members))
		copy(members, b.Members)
		pb.Bindings[i] = &iampb.Binding{
			Role:    b.Role,
			Members: members,
		}
	}
	return pb
}
