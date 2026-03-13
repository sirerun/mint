package azure

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

type mockContainerAppClientForRollback struct {
	revisions    []Revision
	revisionErr  error
	trafficErr   error
	trafficCalls [][]TrafficWeight
}

func (m *mockContainerAppClientForRollback) CreateOrUpdateApp(_ context.Context, _ *CreateOrUpdateAppInput) (*ContainerApp, error) {
	return nil, nil
}

func (m *mockContainerAppClientForRollback) GetApp(_ context.Context, _, _ string) (*ContainerApp, error) {
	return nil, nil
}

func (m *mockContainerAppClientForRollback) ListRevisions(_ context.Context, _, _ string) ([]Revision, error) {
	if m.revisionErr != nil {
		return nil, m.revisionErr
	}
	return m.revisions, nil
}

func (m *mockContainerAppClientForRollback) UpdateTrafficSplit(_ context.Context, _, _ string, traffic []TrafficWeight) error {
	m.trafficCalls = append(m.trafficCalls, traffic)
	return m.trafficErr
}

func TestRollback_Success(t *testing.T) {
	client := &mockContainerAppClientForRollback{
		revisions: []Revision{
			{Name: "my-app--rev3", Active: true, TrafficWeight: 100, CreatedTime: "2026-03-13T10:00:00Z"},
			{Name: "my-app--rev2", Active: true, TrafficWeight: 0, CreatedTime: "2026-03-12T10:00:00Z"},
			{Name: "my-app--rev1", Active: false, TrafficWeight: 0, CreatedTime: "2026-03-11T10:00:00Z"},
		},
	}

	var stderr bytes.Buffer
	result, err := Rollback(context.Background(), client, "my-app", "my-rg", &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.CurrentRevision != "my-app--rev3" {
		t.Errorf("CurrentRevision = %q, want %q", result.CurrentRevision, "my-app--rev3")
	}
	if result.PreviousRevision != "my-app--rev2" {
		t.Errorf("PreviousRevision = %q, want %q", result.PreviousRevision, "my-app--rev2")
	}
	if result.ServiceName != "my-app" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-app")
	}
	if result.ResourceGroup != "my-rg" {
		t.Errorf("ResourceGroup = %q, want %q", result.ResourceGroup, "my-rg")
	}

	// Verify traffic split was called with 100% to previous revision.
	if len(client.trafficCalls) != 1 {
		t.Fatalf("expected 1 UpdateTrafficSplit call, got %d", len(client.trafficCalls))
	}
	tw := client.trafficCalls[0]
	if len(tw) != 1 {
		t.Fatalf("expected 1 traffic weight, got %d", len(tw))
	}
	if tw[0].RevisionName != "my-app--rev2" {
		t.Errorf("traffic RevisionName = %q, want %q", tw[0].RevisionName, "my-app--rev2")
	}
	if tw[0].Weight != 100 {
		t.Errorf("traffic Weight = %d, want 100", tw[0].Weight)
	}

	// Verify stderr output.
	if !strings.Contains(stderr.String(), "Rolling back") {
		t.Errorf("stderr missing 'Rolling back', got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Rollback complete") {
		t.Errorf("stderr missing 'Rollback complete', got %q", stderr.String())
	}
}

func TestRollback_NilStderr(t *testing.T) {
	client := &mockContainerAppClientForRollback{
		revisions: []Revision{
			{Name: "rev2", Active: true, TrafficWeight: 100},
			{Name: "rev1", Active: true, TrafficWeight: 0},
		},
	}

	result, err := Rollback(context.Background(), client, "svc", "rg", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PreviousRevision != "rev1" {
		t.Errorf("PreviousRevision = %q, want %q", result.PreviousRevision, "rev1")
	}
}

func TestRollback_OnlyOneRevision(t *testing.T) {
	client := &mockContainerAppClientForRollback{
		revisions: []Revision{
			{Name: "my-app--rev1", Active: true, TrafficWeight: 100},
		},
	}

	_, err := Rollback(context.Background(), client, "my-app", "my-rg", nil)
	if err == nil {
		t.Fatal("expected error for single revision, got nil")
	}
	want := "only 1 revision found: rollback requires at least 2 revisions"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestRollback_NoRevisions(t *testing.T) {
	client := &mockContainerAppClientForRollback{
		revisions: []Revision{},
	}

	_, err := Rollback(context.Background(), client, "my-app", "my-rg", nil)
	if err == nil {
		t.Fatal("expected error for no revisions, got nil")
	}
	want := "no revisions found: rollback requires at least 2 revisions"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestRollback_ListRevisionsError(t *testing.T) {
	listErr := errors.New("network failure")
	client := &mockContainerAppClientForRollback{
		revisionErr: listErr,
	}

	_, err := Rollback(context.Background(), client, "svc", "rg", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, listErr) {
		t.Errorf("expected wrapped listErr, got: %v", err)
	}
}

func TestRollback_UpdateTrafficSplitError(t *testing.T) {
	trafficErr := errors.New("permission denied")
	client := &mockContainerAppClientForRollback{
		revisions: []Revision{
			{Name: "rev2", Active: true, TrafficWeight: 100},
			{Name: "rev1", Active: true, TrafficWeight: 0},
		},
		trafficErr: trafficErr,
	}

	_, err := Rollback(context.Background(), client, "svc", "rg", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, trafficErr) {
		t.Errorf("expected wrapped trafficErr, got: %v", err)
	}
}
