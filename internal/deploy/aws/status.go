package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// jsonMarshalIndent is a variable to allow tests to inject marshal failures.
var jsonMarshalIndent = json.MarshalIndent

// StatusClient abstracts operations needed for the AWS status command.
type StatusClient interface {
	DescribeService(ctx context.Context, cluster, serviceName string) (*ServiceStatus, error)
	DescribeTargetHealth(ctx context.Context, targetGroupARN string) ([]TargetHealthStatus, error)
}

// ServiceStatus holds the status of an ECS Fargate service.
type ServiceStatus struct {
	ServiceName       string
	ClusterARN        string
	TaskDefinitionARN string
	Status            string
	DesiredCount      int
	RunningCount      int
	PendingCount      int
}

// TargetHealthStatus holds the health status of a target in an ALB target group.
type TargetHealthStatus struct {
	TargetID    string
	State       string
	Description string
}

// StatusResult is the output of the AWS status command.
type StatusResult struct {
	ServiceName       string       `json:"service_name"`
	ClusterARN        string       `json:"cluster_arn"`
	TaskDefinitionARN string       `json:"task_definition_arn"`
	Status            string       `json:"status"`
	DesiredCount      int          `json:"desired_count"`
	RunningCount      int          `json:"running_count"`
	PendingCount      int          `json:"pending_count"`
	Targets           []TargetInfo `json:"targets"`
}

// TargetInfo is a target in the status output.
type TargetInfo struct {
	TargetID    string `json:"target_id"`
	State       string `json:"state"`
	Description string `json:"description,omitempty"`
}

// GetStatus retrieves the status of an ECS Fargate service and its ALB target health.
// If target health retrieval fails, a partial result is returned with the service
// information and the target health error.
func GetStatus(ctx context.Context, client StatusClient, cluster, serviceName, targetGroupARN string) (*StatusResult, error) {
	svc, err := client.DescribeService(ctx, cluster, serviceName)
	if err != nil {
		return nil, fmt.Errorf("describing service %s: %w", serviceName, err)
	}

	result := &StatusResult{
		ServiceName:       svc.ServiceName,
		ClusterARN:        svc.ClusterARN,
		TaskDefinitionARN: svc.TaskDefinitionARN,
		Status:            svc.Status,
		DesiredCount:      svc.DesiredCount,
		RunningCount:      svc.RunningCount,
		PendingCount:      svc.PendingCount,
	}

	if targetGroupARN == "" {
		return result, nil
	}

	targets, err := client.DescribeTargetHealth(ctx, targetGroupARN)
	if err != nil {
		// Return partial result with service info when target health fails.
		return result, fmt.Errorf("describing target health: %w", err)
	}

	result.Targets = make([]TargetInfo, len(targets))
	for i, t := range targets {
		result.Targets[i] = TargetInfo(t)
	}

	return result, nil
}

// FormatStatus formats a StatusResult as either JSON or human-readable text.
func FormatStatus(result *StatusResult, jsonOutput bool) string {
	if jsonOutput {
		data, err := jsonMarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		return string(data)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Service:         %s\n", result.ServiceName)
	fmt.Fprintf(&b, "Cluster:         %s\n", result.ClusterARN)
	fmt.Fprintf(&b, "Task Definition: %s\n", result.TaskDefinitionARN)
	fmt.Fprintf(&b, "Status:          %s\n", result.Status)
	fmt.Fprintf(&b, "Desired:         %d\n", result.DesiredCount)
	fmt.Fprintf(&b, "Running:         %d\n", result.RunningCount)
	fmt.Fprintf(&b, "Pending:         %d\n", result.PendingCount)

	if len(result.Targets) > 0 {
		fmt.Fprintf(&b, "\nTargets:\n")
		fmt.Fprintf(&b, "  %-40s %-12s %s\n", "TARGET", "STATE", "DESCRIPTION")
		for _, t := range result.Targets {
			fmt.Fprintf(&b, "  %-40s %-12s %s\n", t.TargetID, t.State, t.Description)
		}
	}

	return b.String()
}
