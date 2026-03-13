package aws

import (
	"context"
	"errors"
	"fmt"
)

// RollbackClient abstracts ECS operations needed for rolling back a service
// to a previous task definition revision.
type RollbackClient interface {
	// ListTaskDefinitions returns task definition ARNs for the given family,
	// ordered newest-first.
	ListTaskDefinitions(ctx context.Context, family string) ([]string, error)

	// UpdateService updates an ECS service to use a different task definition.
	UpdateService(ctx context.Context, input *UpdateECSServiceInput) (*ECSService, error)

	// WaitForStableService blocks until the service reaches a steady state.
	WaitForStableService(ctx context.Context, cluster, serviceName string) error
}

// RollbackResult contains the outcome of a rollback operation.
type RollbackResult struct {
	PreviousTaskDef string
	CurrentTaskDef  string
	ServiceName     string
	ClusterARN      string
}

// Rollback updates an ECS Fargate service to use the previous task definition
// revision. It returns an error if fewer than 2 revisions exist in the family.
func Rollback(ctx context.Context, client RollbackClient, cluster, serviceName, family string) (*RollbackResult, error) {
	taskDefs, err := client.ListTaskDefinitions(ctx, family)
	if err != nil {
		return nil, fmt.Errorf("list task definitions for family %s: %w", family, err)
	}

	if len(taskDefs) == 0 {
		return nil, errors.New("no task definitions found: rollback requires at least 2 revisions")
	}
	if len(taskDefs) < 2 {
		return nil, errors.New("only 1 task definition found: rollback requires at least 2 revisions")
	}

	current := taskDefs[0]
	previous := taskDefs[1]

	_, err = client.UpdateService(ctx, &UpdateECSServiceInput{
		Cluster:           cluster,
		ServiceName:       serviceName,
		TaskDefinitionARN: previous,
	})
	if err != nil {
		return nil, fmt.Errorf("update service %s to task definition %s: %w", serviceName, previous, err)
	}

	if err := client.WaitForStableService(ctx, cluster, serviceName); err != nil {
		return nil, fmt.Errorf("wait for stable service %s: %w", serviceName, err)
	}

	return &RollbackResult{
		PreviousTaskDef: previous,
		CurrentTaskDef:  current,
		ServiceName:     serviceName,
		ClusterARN:      cluster,
	}, nil
}
