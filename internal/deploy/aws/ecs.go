package aws

import (
	"context"
	"errors"
)

// ErrServiceNotFound indicates that an ECS service was not found.
var ErrServiceNotFound = errors.New("ecs: service not found")

// ECSClient abstracts Amazon ECS Fargate operations.
type ECSClient interface {
	// CreateCluster creates an ECS cluster if it does not already exist.
	CreateCluster(ctx context.Context, clusterName string) (*Cluster, error)

	// DescribeServices returns metadata for services in a cluster.
	// Returns ErrServiceNotFound if the service does not exist.
	DescribeServices(ctx context.Context, input *DescribeServicesInput) ([]ECSService, error)

	// RegisterTaskDefinition registers a new task definition revision.
	RegisterTaskDefinition(ctx context.Context, input *RegisterTaskDefinitionInput) (*TaskDefinition, error)

	// CreateService creates a new ECS Fargate service.
	CreateService(ctx context.Context, input *CreateECSServiceInput) (*ECSService, error)

	// UpdateService updates an existing ECS Fargate service.
	UpdateService(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error)

	// DescribeTasks returns the status of tasks in a cluster.
	DescribeTasks(ctx context.Context, cluster string, taskARNs []string) ([]Task, error)
}

// Cluster represents an ECS cluster.
type Cluster struct {
	ClusterARN  string
	ClusterName string
}

// DescribeServicesInput holds parameters for describing ECS services.
type DescribeServicesInput struct {
	Cluster     string
	ServiceName string
}

// ECSService represents an ECS Fargate service.
type ECSService struct {
	ServiceARN        string
	ServiceName       string
	Status            string
	ClusterARN        string
	TaskDefinitionARN string
	DesiredCount      int
	RunningCount      int
}

// RegisterTaskDefinitionInput holds parameters for registering a task definition.
type RegisterTaskDefinitionInput struct {
	Family           string
	ImageURI         string
	ContainerName    string
	Port             int
	CPU              string
	Memory           string
	EnvVars          map[string]string
	SecretARNs       []string
	ExecutionRoleARN string
	TaskRoleARN      string
	Args             []string
}

// TaskDefinition represents a registered ECS task definition.
type TaskDefinition struct {
	TaskDefinitionARN string
	Family            string
	Revision          int
}

// CreateECSServiceInput holds parameters for creating an ECS Fargate service.
type CreateECSServiceInput struct {
	Cluster           string
	ServiceName       string
	TaskDefinitionARN string
	DesiredCount      int
	SubnetIDs         []string
	SecurityGroupIDs  []string
	AssignPublicIP    bool
	TargetGroupARN    string
}

// UpdateECSServiceInput holds parameters for updating an ECS Fargate service.
type UpdateECSServiceInput struct {
	Cluster           string
	ServiceName       string
	TaskDefinitionARN string
	DesiredCount      int
}

// Task represents the status of an ECS task.
type Task struct {
	TaskARN    string
	LastStatus string
}
