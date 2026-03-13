package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type mockStatusClient struct {
	service    *ServiceStatus
	targets    []TargetHealthStatus
	serviceErr error
	targetErr  error
}

func (m *mockStatusClient) DescribeService(_ context.Context, _, _ string) (*ServiceStatus, error) {
	if m.serviceErr != nil {
		return nil, m.serviceErr
	}
	return m.service, nil
}

func (m *mockStatusClient) DescribeTargetHealth(_ context.Context, _ string) ([]TargetHealthStatus, error) {
	if m.targetErr != nil {
		return nil, m.targetErr
	}
	return m.targets, nil
}

func TestGetStatus_Success(t *testing.T) {
	client := &mockStatusClient{
		service: &ServiceStatus{
			ServiceName:       "my-svc",
			ClusterARN:        "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster",
			TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/my-svc:5",
			Status:            "ACTIVE",
			DesiredCount:      3,
			RunningCount:      3,
			PendingCount:      0,
		},
		targets: []TargetHealthStatus{
			{TargetID: "10.0.1.10", State: "healthy", Description: ""},
			{TargetID: "10.0.1.11", State: "healthy", Description: ""},
			{TargetID: "10.0.1.12", State: "draining", Description: "Target deregistration in progress"},
		},
	}

	result, err := GetStatus(context.Background(), client, "my-cluster", "my-svc", "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-tg/abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ServiceName != "my-svc" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-svc")
	}
	if result.ClusterARN != "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster" {
		t.Errorf("ClusterARN = %q, want %q", result.ClusterARN, "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster")
	}
	if result.TaskDefinitionARN != "arn:aws:ecs:us-east-1:123456789012:task-definition/my-svc:5" {
		t.Errorf("TaskDefinitionARN = %q, want %q", result.TaskDefinitionARN, "arn:aws:ecs:us-east-1:123456789012:task-definition/my-svc:5")
	}
	if result.Status != "ACTIVE" {
		t.Errorf("Status = %q, want %q", result.Status, "ACTIVE")
	}
	if result.DesiredCount != 3 {
		t.Errorf("DesiredCount = %d, want 3", result.DesiredCount)
	}
	if result.RunningCount != 3 {
		t.Errorf("RunningCount = %d, want 3", result.RunningCount)
	}
	if result.PendingCount != 0 {
		t.Errorf("PendingCount = %d, want 0", result.PendingCount)
	}
	if len(result.Targets) != 3 {
		t.Fatalf("Targets count = %d, want 3", len(result.Targets))
	}
	if result.Targets[0].TargetID != "10.0.1.10" {
		t.Errorf("Targets[0].TargetID = %q, want %q", result.Targets[0].TargetID, "10.0.1.10")
	}
	if result.Targets[0].State != "healthy" {
		t.Errorf("Targets[0].State = %q, want %q", result.Targets[0].State, "healthy")
	}
	if result.Targets[2].State != "draining" {
		t.Errorf("Targets[2].State = %q, want %q", result.Targets[2].State, "draining")
	}
	if result.Targets[2].Description != "Target deregistration in progress" {
		t.Errorf("Targets[2].Description = %q, want %q", result.Targets[2].Description, "Target deregistration in progress")
	}
}

func TestGetStatus_ServiceNotFound(t *testing.T) {
	client := &mockStatusClient{
		serviceErr: fmt.Errorf("service not found"),
	}

	_, err := GetStatus(context.Background(), client, "my-cluster", "missing-svc", "arn:tg")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "describing service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "describing service")
	}
	if !strings.Contains(err.Error(), "service not found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "service not found")
	}
}

func TestGetStatus_TargetHealthFailure_PartialResult(t *testing.T) {
	client := &mockStatusClient{
		service: &ServiceStatus{
			ServiceName:       "my-svc",
			ClusterARN:        "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster",
			TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/my-svc:3",
			Status:            "ACTIVE",
			DesiredCount:      2,
			RunningCount:      2,
			PendingCount:      0,
		},
		targetErr: fmt.Errorf("access denied"),
	}

	result, err := GetStatus(context.Background(), client, "my-cluster", "my-svc", "arn:tg")
	if err == nil {
		t.Fatal("expected error for target health failure")
	}
	if !strings.Contains(err.Error(), "describing target health") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "describing target health")
	}
	// Partial result should still contain service info.
	if result == nil {
		t.Fatal("expected partial result, got nil")
	}
	if result.ServiceName != "my-svc" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-svc")
	}
	if result.DesiredCount != 2 {
		t.Errorf("DesiredCount = %d, want 2", result.DesiredCount)
	}
	if result.Targets != nil {
		t.Errorf("Targets = %v, want nil on target health failure", result.Targets)
	}
}

func TestFormatStatus_HumanReadable(t *testing.T) {
	result := &StatusResult{
		ServiceName:       "my-svc",
		ClusterARN:        "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster",
		TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/my-svc:5",
		Status:            "ACTIVE",
		DesiredCount:      3,
		RunningCount:      2,
		PendingCount:      1,
		Targets: []TargetInfo{
			{TargetID: "10.0.1.10", State: "healthy", Description: ""},
			{TargetID: "10.0.1.11", State: "unhealthy", Description: "Health check failed"},
		},
	}

	output := FormatStatus(result, false)

	checks := []string{
		"my-svc",
		"my-cluster",
		"task-definition/my-svc:5",
		"ACTIVE",
		"3",
		"2",
		"1",
		"TARGET",
		"STATE",
		"DESCRIPTION",
		"10.0.1.10",
		"healthy",
		"10.0.1.11",
		"unhealthy",
		"Health check failed",
	}

	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("text output missing %q\nOutput:\n%s", want, output)
		}
	}
}

func TestFormatStatus_JSON(t *testing.T) {
	result := &StatusResult{
		ServiceName:       "test-svc",
		ClusterARN:        "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster",
		TaskDefinitionARN: "arn:aws:ecs:us-east-1:123456789012:task-definition/test-svc:1",
		Status:            "ACTIVE",
		DesiredCount:      2,
		RunningCount:      2,
		PendingCount:      0,
		Targets: []TargetInfo{
			{TargetID: "10.0.1.10", State: "healthy"},
		},
	}

	output := FormatStatus(result, true)

	var parsed StatusResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\nOutput:\n%s", err, output)
	}

	if parsed.ServiceName != "test-svc" {
		t.Errorf("parsed ServiceName = %q, want %q", parsed.ServiceName, "test-svc")
	}
	if parsed.ClusterARN != "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster" {
		t.Errorf("parsed ClusterARN = %q, want %q", parsed.ClusterARN, "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster")
	}
	if parsed.DesiredCount != 2 {
		t.Errorf("parsed DesiredCount = %d, want 2", parsed.DesiredCount)
	}
	if parsed.RunningCount != 2 {
		t.Errorf("parsed RunningCount = %d, want 2", parsed.RunningCount)
	}
	if len(parsed.Targets) != 1 {
		t.Fatalf("parsed Targets count = %d, want 1", len(parsed.Targets))
	}
	if parsed.Targets[0].TargetID != "10.0.1.10" {
		t.Errorf("parsed Targets[0].TargetID = %q, want %q", parsed.Targets[0].TargetID, "10.0.1.10")
	}
	if parsed.Targets[0].State != "healthy" {
		t.Errorf("parsed Targets[0].State = %q, want %q", parsed.Targets[0].State, "healthy")
	}
}
