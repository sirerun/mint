package aws

import (
	"context"
	"errors"
)

// ErrRoleNotFound indicates that an IAM role was not found.
var ErrRoleNotFound = errors.New("iam: role not found")

// IAMClient abstracts AWS IAM operations.
type IAMClient interface {
	// GetRole returns an existing IAM role.
	// Returns ErrRoleNotFound if the role does not exist.
	GetRole(ctx context.Context, roleName string) (*Role, error)

	// CreateRole creates a new IAM role.
	CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error)

	// AttachRolePolicy attaches a managed policy to a role.
	AttachRolePolicy(ctx context.Context, roleName, policyARN string) error
}

// Role represents an IAM role.
type Role struct {
	ARN      string
	RoleName string
}

// CreateRoleInput holds parameters for creating an IAM role.
type CreateRoleInput struct {
	RoleName                 string
	AssumeRolePolicyDocument string
	Description              string
}
