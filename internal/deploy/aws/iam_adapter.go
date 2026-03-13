package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// IAMAdapter implements IAMClient using the AWS SDK v2.
type IAMAdapter struct {
	client *iam.Client
}

var _ IAMClient = (*IAMAdapter)(nil)

// NewIAMAdapter creates a new adapter backed by the AWS IAM SDK client.
func NewIAMAdapter(cfg aws.Config) *IAMAdapter {
	return &IAMAdapter{client: iam.NewFromConfig(cfg)}
}

// GetRole returns an existing IAM role.
func (a *IAMAdapter) GetRole(ctx context.Context, roleName string) (*Role, error) {
	out, err := a.client.GetRole(ctx, &iam.GetRoleInput{
		RoleName: &roleName,
	})
	if err != nil {
		var notFound *types.NoSuchEntityException
		if errors.As(err, &notFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &Role{
		ARN:      derefStr(out.Role.Arn),
		RoleName: derefStr(out.Role.RoleName),
	}, nil
}

// CreateRole creates a new IAM role.
func (a *IAMAdapter) CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error) {
	out, err := a.client.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 &input.RoleName,
		AssumeRolePolicyDocument: &input.AssumeRolePolicyDocument,
		Description:              &input.Description,
	})
	if err != nil {
		return nil, err
	}
	return &Role{
		ARN:      derefStr(out.Role.Arn),
		RoleName: derefStr(out.Role.RoleName),
	}, nil
}

// AttachRolePolicy attaches a managed policy to a role.
func (a *IAMAdapter) AttachRolePolicy(ctx context.Context, roleName, policyARN string) error {
	_, err := a.client.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  &roleName,
		PolicyArn: &policyARN,
	})
	return err
}

// TaskRoles holds the ARNs for ECS task execution and task roles.
type TaskRoles struct {
	ExecutionRoleARN string
	TaskRoleARN      string
}

const ecsTaskExecutionPolicyARN = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"

// EnsureTaskRoles ensures that both the ECS execution role and task role exist,
// creating them if necessary. It returns both role ARNs.
func EnsureTaskRoles(ctx context.Context, client IAMClient, serviceName string) (*TaskRoles, error) {
	trustPolicy, err := ecsTrustPolicy()
	if err != nil {
		return nil, fmt.Errorf("marshal trust policy: %w", err)
	}

	execRoleName := "mint-ecs-execution-" + serviceName
	execRole, err := ensureRole(ctx, client, &CreateRoleInput{
		RoleName:                 execRoleName,
		AssumeRolePolicyDocument: trustPolicy,
		Description:              "Mint ECS execution role for " + serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure execution role: %w", err)
	}

	if err := client.AttachRolePolicy(ctx, execRoleName, ecsTaskExecutionPolicyARN); err != nil {
		return nil, fmt.Errorf("attach execution policy: %w", err)
	}

	taskRoleName := "mint-ecs-task-" + serviceName
	taskRole, err := ensureRole(ctx, client, &CreateRoleInput{
		RoleName:                 taskRoleName,
		AssumeRolePolicyDocument: trustPolicy,
		Description:              "Mint ECS task role for " + serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure task role: %w", err)
	}

	return &TaskRoles{
		ExecutionRoleARN: execRole.ARN,
		TaskRoleARN:      taskRole.ARN,
	}, nil
}

func ensureRole(ctx context.Context, client IAMClient, input *CreateRoleInput) (*Role, error) {
	role, err := client.GetRole(ctx, input.RoleName)
	if err == nil {
		return role, nil
	}
	if !errors.Is(err, ErrRoleNotFound) {
		return nil, err
	}
	return client.CreateRole(ctx, input)
}

func ecsTrustPolicy() (string, error) {
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Effect": "Allow",
				"Principal": map[string]string{
					"Service": "ecs-tasks.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
