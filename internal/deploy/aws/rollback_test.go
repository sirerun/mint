package aws

import (
	"context"
	"errors"
	"testing"
)

// mockRollbackClient implements RollbackClient for testing.
type mockRollbackClient struct {
	taskDefs      []string
	listErr       error
	updateErr     error
	waitErr       error
	updateCalls   []UpdateECSServiceInput
	waitCalls     []waitCall
}

type waitCall struct {
	cluster     string
	serviceName string
}

func (m *mockRollbackClient) ListTaskDefinitions(_ context.Context, _ string) ([]string, error) {
	return m.taskDefs, m.listErr
}

func (m *mockRollbackClient) UpdateService(_ context.Context, input *UpdateECSServiceInput) (*ECSService, error) {
	m.updateCalls = append(m.updateCalls, *input)
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return &ECSService{ServiceName: input.ServiceName}, nil
}

func (m *mockRollbackClient) WaitForStableService(_ context.Context, cluster, serviceName string) error {
	m.waitCalls = append(m.waitCalls, waitCall{cluster: cluster, serviceName: serviceName})
	return m.waitErr
}

func TestRollback_Success(t *testing.T) {
	client := &mockRollbackClient{
		taskDefs: []string{
			"arn:aws:ecs:us-east-1:123456:task-definition/my-svc:3",
			"arn:aws:ecs:us-east-1:123456:task-definition/my-svc:2",
			"arn:aws:ecs:us-east-1:123456:task-definition/my-svc:1",
		},
	}

	result, err := Rollback(context.Background(), client, "my-cluster", "my-svc", "my-svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CurrentTaskDef != "arn:aws:ecs:us-east-1:123456:task-definition/my-svc:3" {
		t.Errorf("CurrentTaskDef = %q, want revision :3", result.CurrentTaskDef)
	}
	if result.PreviousTaskDef != "arn:aws:ecs:us-east-1:123456:task-definition/my-svc:2" {
		t.Errorf("PreviousTaskDef = %q, want revision :2", result.PreviousTaskDef)
	}
	if result.ServiceName != "my-svc" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-svc")
	}
	if result.ClusterARN != "my-cluster" {
		t.Errorf("ClusterARN = %q, want %q", result.ClusterARN, "my-cluster")
	}

	// Verify UpdateService was called with the previous task definition.
	if len(client.updateCalls) != 1 {
		t.Fatalf("expected 1 UpdateService call, got %d", len(client.updateCalls))
	}
	uc := client.updateCalls[0]
	if uc.TaskDefinitionARN != "arn:aws:ecs:us-east-1:123456:task-definition/my-svc:2" {
		t.Errorf("UpdateService TaskDefinitionARN = %q, want revision :2", uc.TaskDefinitionARN)
	}
	if uc.Cluster != "my-cluster" {
		t.Errorf("UpdateService Cluster = %q, want %q", uc.Cluster, "my-cluster")
	}

	// Verify WaitForStableService was called.
	if len(client.waitCalls) != 1 {
		t.Fatalf("expected 1 WaitForStableService call, got %d", len(client.waitCalls))
	}
	wc := client.waitCalls[0]
	if wc.cluster != "my-cluster" || wc.serviceName != "my-svc" {
		t.Errorf("WaitForStableService called with (%q, %q), want (%q, %q)", wc.cluster, wc.serviceName, "my-cluster", "my-svc")
	}
}

func TestRollback_OnlyOneRevision(t *testing.T) {
	client := &mockRollbackClient{
		taskDefs: []string{
			"arn:aws:ecs:us-east-1:123456:task-definition/my-svc:1",
		},
	}

	_, err := Rollback(context.Background(), client, "cluster", "svc", "family")
	if err == nil {
		t.Fatal("expected error for single revision, got nil")
	}
	want := "only 1 task definition found: rollback requires at least 2 revisions"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestRollback_NoRevisions(t *testing.T) {
	client := &mockRollbackClient{
		taskDefs: []string{},
	}

	_, err := Rollback(context.Background(), client, "cluster", "svc", "family")
	if err == nil {
		t.Fatal("expected error for no revisions, got nil")
	}
	want := "no task definitions found: rollback requires at least 2 revisions"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestRollback_ListTaskDefinitionsError(t *testing.T) {
	listErr := errors.New("network failure")
	client := &mockRollbackClient{
		listErr: listErr,
	}

	_, err := Rollback(context.Background(), client, "cluster", "svc", "family")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, listErr) {
		t.Errorf("expected wrapped listErr, got: %v", err)
	}
}

func TestRollback_UpdateServiceError(t *testing.T) {
	updateErr := errors.New("permission denied")
	client := &mockRollbackClient{
		taskDefs: []string{
			"arn:aws:ecs:us-east-1:123456:task-definition/svc:2",
			"arn:aws:ecs:us-east-1:123456:task-definition/svc:1",
		},
		updateErr: updateErr,
	}

	_, err := Rollback(context.Background(), client, "cluster", "svc", "family")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, updateErr) {
		t.Errorf("expected wrapped updateErr, got: %v", err)
	}
}

func TestRollback_WaitForStableServiceError(t *testing.T) {
	waitErr := errors.New("service did not stabilize")
	client := &mockRollbackClient{
		taskDefs: []string{
			"arn:aws:ecs:us-east-1:123456:task-definition/svc:2",
			"arn:aws:ecs:us-east-1:123456:task-definition/svc:1",
		},
		waitErr: waitErr,
	}

	_, err := Rollback(context.Background(), client, "cluster", "svc", "family")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, waitErr) {
		t.Errorf("expected wrapped waitErr, got: %v", err)
	}

	// Verify that UpdateService was still called (rollback was initiated).
	if len(client.updateCalls) != 1 {
		t.Errorf("expected 1 UpdateService call even though wait failed, got %d", len(client.updateCalls))
	}
}
