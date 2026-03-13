package azure

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type mockContainerAppClientForCanary struct {
	revisions    []Revision
	revisionErr  error
	trafficErr   error
	trafficCalls [][]TrafficWeight
}

func (m *mockContainerAppClientForCanary) CreateOrUpdateApp(_ context.Context, _ *CreateOrUpdateAppInput) (*ContainerApp, error) {
	return nil, nil
}

func (m *mockContainerAppClientForCanary) GetApp(_ context.Context, _, _ string) (*ContainerApp, error) {
	return nil, nil
}

func (m *mockContainerAppClientForCanary) ListRevisions(_ context.Context, _, _ string) ([]Revision, error) {
	if m.revisionErr != nil {
		return nil, m.revisionErr
	}
	return m.revisions, nil
}

func (m *mockContainerAppClientForCanary) UpdateTrafficSplit(_ context.Context, _, _ string, traffic []TrafficWeight) error {
	m.trafficCalls = append(m.trafficCalls, traffic)
	return m.trafficErr
}

func TestSetCanaryTraffic(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{
			revisions: []Revision{
				{Name: "my-app--rev2", Active: true, TrafficWeight: 100},
				{Name: "my-app--rev1", Active: true, TrafficWeight: 0},
			},
		}

		err := SetCanaryTraffic(context.Background(), client, "my-rg", "my-app", "my-app--canary", 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(client.trafficCalls) != 1 {
			t.Fatalf("expected 1 UpdateTrafficSplit call, got %d", len(client.trafficCalls))
		}
		tw := client.trafficCalls[0]
		if len(tw) != 2 {
			t.Fatalf("expected 2 traffic weights, got %d", len(tw))
		}
		// Stable gets 90%, canary gets 10%.
		if tw[0].RevisionName != "my-app--rev2" {
			t.Errorf("stable revision = %q, want %q", tw[0].RevisionName, "my-app--rev2")
		}
		if tw[0].Weight != 90 {
			t.Errorf("stable weight = %d, want 90", tw[0].Weight)
		}
		if tw[1].RevisionName != "my-app--canary" {
			t.Errorf("canary revision = %q, want %q", tw[1].RevisionName, "my-app--canary")
		}
		if tw[1].Weight != 10 {
			t.Errorf("canary weight = %d, want 10", tw[1].Weight)
		}
	})

	t.Run("invalid percent zero", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := SetCanaryTraffic(context.Background(), client, "rg", "svc", "canary", 0)
		if err == nil {
			t.Fatal("expected error for canary percent 0")
		}
		if !strings.Contains(err.Error(), "canary percent must be between 1 and 99") {
			t.Errorf("error = %q, want canary percent validation error", err.Error())
		}
	})

	t.Run("invalid percent 100", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := SetCanaryTraffic(context.Background(), client, "rg", "svc", "canary", 100)
		if err == nil {
			t.Fatal("expected error for canary percent 100")
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := SetCanaryTraffic(context.Background(), client, "rg", "", "canary", 10)
		if err == nil {
			t.Fatal("expected error for empty service name")
		}
	})

	t.Run("empty resource group", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := SetCanaryTraffic(context.Background(), client, "", "svc", "canary", 10)
		if err == nil {
			t.Fatal("expected error for empty resource group")
		}
	})

	t.Run("empty canary revision", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := SetCanaryTraffic(context.Background(), client, "rg", "svc", "", 10)
		if err == nil {
			t.Fatal("expected error for empty canary revision")
		}
	})

	t.Run("list revisions fails", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{
			revisionErr: errors.New("network timeout"),
		}
		err := SetCanaryTraffic(context.Background(), client, "rg", "svc", "canary", 10)
		if err == nil {
			t.Fatal("expected error when ListRevisions fails")
		}
		if !strings.Contains(err.Error(), "list revisions") {
			t.Errorf("error = %q, want it to contain %q", err.Error(), "list revisions")
		}
	})

	t.Run("no current revision found", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{
			revisions: []Revision{
				{Name: "canary", Active: true, TrafficWeight: 100},
			},
		}
		err := SetCanaryTraffic(context.Background(), client, "rg", "svc", "canary", 10)
		if err == nil {
			t.Fatal("expected error when no current revision found")
		}
		if !strings.Contains(err.Error(), "no current revision found") {
			t.Errorf("error = %q, want it to contain %q", err.Error(), "no current revision found")
		}
	})

	t.Run("update traffic split fails", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{
			revisions: []Revision{
				{Name: "stable", Active: true, TrafficWeight: 100},
			},
			trafficErr: errors.New("permission denied"),
		}
		err := SetCanaryTraffic(context.Background(), client, "rg", "svc", "canary", 10)
		if err == nil {
			t.Fatal("expected error when UpdateTrafficSplit fails")
		}
	})
}

func TestPromoteCanary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}

		err := PromoteCanary(context.Background(), client, "my-rg", "my-app", "my-app--canary")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(client.trafficCalls) != 1 {
			t.Fatalf("expected 1 UpdateTrafficSplit call, got %d", len(client.trafficCalls))
		}
		tw := client.trafficCalls[0]
		if len(tw) != 1 {
			t.Fatalf("expected 1 traffic weight, got %d", len(tw))
		}
		if tw[0].RevisionName != "my-app--canary" {
			t.Errorf("revision = %q, want %q", tw[0].RevisionName, "my-app--canary")
		}
		if tw[0].Weight != 100 {
			t.Errorf("weight = %d, want 100", tw[0].Weight)
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := PromoteCanary(context.Background(), client, "rg", "", "canary")
		if err == nil {
			t.Fatal("expected error for empty service name")
		}
	})

	t.Run("empty resource group", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := PromoteCanary(context.Background(), client, "", "svc", "canary")
		if err == nil {
			t.Fatal("expected error for empty resource group")
		}
	})

	t.Run("empty canary revision", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{}
		err := PromoteCanary(context.Background(), client, "rg", "svc", "")
		if err == nil {
			t.Fatal("expected error for empty canary revision")
		}
	})

	t.Run("update traffic split fails", func(t *testing.T) {
		client := &mockContainerAppClientForCanary{
			trafficErr: errors.New("permission denied"),
		}
		err := PromoteCanary(context.Background(), client, "rg", "svc", "canary")
		if err == nil {
			t.Fatal("expected error when UpdateTrafficSplit fails")
		}
	})
}
