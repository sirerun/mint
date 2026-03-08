package gcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockRevisionClient implements RevisionClient for testing.
type mockRevisionClient struct {
	revisions    []Revision
	listErr      error
	trafficErr   error
	trafficCalls []trafficCall
}

type trafficCall struct {
	serviceName  string
	revisionName string
	percent      int
}

func (m *mockRevisionClient) ListRevisions(_ context.Context, _ string) ([]Revision, error) {
	return m.revisions, m.listErr
}

func (m *mockRevisionClient) UpdateTraffic(_ context.Context, serviceName, revisionName string, percent int) error {
	m.trafficCalls = append(m.trafficCalls, trafficCall{
		serviceName:  serviceName,
		revisionName: revisionName,
		percent:      percent,
	})
	return m.trafficErr
}

func TestRollback_Success(t *testing.T) {
	now := time.Now()
	client := &mockRevisionClient{
		revisions: []Revision{
			{Name: "rev-002", CreateTime: now, Active: true},
			{Name: "rev-001", CreateTime: now.Add(-time.Hour), Active: false},
		},
	}

	result, err := Rollback(context.Background(), client, "my-project", "us-central1", "my-svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PreviousRevision != "rev-001" {
		t.Errorf("PreviousRevision = %q, want %q", result.PreviousRevision, "rev-001")
	}
	if result.CurrentRevision != "rev-002" {
		t.Errorf("CurrentRevision = %q, want %q", result.CurrentRevision, "rev-002")
	}
	wantService := "projects/my-project/locations/us-central1/services/my-svc"
	if result.ServiceName != wantService {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, wantService)
	}

	if len(client.trafficCalls) != 1 {
		t.Fatalf("expected 1 UpdateTraffic call, got %d", len(client.trafficCalls))
	}
	tc := client.trafficCalls[0]
	if tc.revisionName != "rev-001" {
		t.Errorf("UpdateTraffic revisionName = %q, want %q", tc.revisionName, "rev-001")
	}
	if tc.percent != 100 {
		t.Errorf("UpdateTraffic percent = %d, want 100", tc.percent)
	}
}

func TestRollback_OneRevision(t *testing.T) {
	client := &mockRevisionClient{
		revisions: []Revision{
			{Name: "rev-001", CreateTime: time.Now(), Active: true},
		},
	}

	_, err := Rollback(context.Background(), client, "p", "r", "s")
	if err == nil {
		t.Fatal("expected error for single revision, got nil")
	}
	if got := err.Error(); got != "only 1 revision found: rollback requires at least 2 revisions" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestRollback_NoRevisions(t *testing.T) {
	client := &mockRevisionClient{
		revisions: []Revision{},
	}

	_, err := Rollback(context.Background(), client, "p", "r", "s")
	if err == nil {
		t.Fatal("expected error for no revisions, got nil")
	}
	if got := err.Error(); got != "no revisions found: rollback requires at least 2 revisions" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestRollback_ListRevisionsError(t *testing.T) {
	listErr := errors.New("network failure")
	client := &mockRevisionClient{
		listErr: listErr,
	}

	_, err := Rollback(context.Background(), client, "p", "r", "s")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, listErr) {
		t.Errorf("expected wrapped listErr, got: %v", err)
	}
}

func TestRollback_UpdateTrafficError(t *testing.T) {
	now := time.Now()
	trafficErr := errors.New("permission denied")
	client := &mockRevisionClient{
		revisions: []Revision{
			{Name: "rev-002", CreateTime: now, Active: true},
			{Name: "rev-001", CreateTime: now.Add(-time.Hour), Active: false},
		},
		trafficErr: trafficErr,
	}

	_, err := Rollback(context.Background(), client, "p", "r", "s")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, trafficErr) {
		t.Errorf("expected wrapped trafficErr, got: %v", err)
	}
}
