package gcp

import "context"

// IAMPolicyClient manages IAM policies for Cloud Run services.
type IAMPolicyClient interface {
	// ConfigureIAMPolicy sets the IAM policy on a Cloud Run service.
	// If allowUnauthenticated is true, it grants allUsers the invoker role.
	ConfigureIAMPolicy(ctx context.Context, projectID, region, serviceName string, allowUnauthenticated bool) error
}
