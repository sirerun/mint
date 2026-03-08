package gcp

import (
	"context"
	"errors"
	"testing"
)

// mockTrafficClient implements TrafficClient for testing.
type mockTrafficClient struct {
	targets    []TrafficTarget
	getErr     error
	setErr     error
	setTargets []TrafficTarget // captures the last SetTraffic call
}

func (m *mockTrafficClient) GetTraffic(_ context.Context, _ string) ([]TrafficTarget, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.targets, nil
}

func (m *mockTrafficClient) SetTraffic(_ context.Context, _ string, targets []TrafficTarget) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.setTargets = targets
	return nil
}

func TestSetCanaryTraffic(t *testing.T) {
	t.Run("splits traffic correctly", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{
				{RevisionName: "svc-00001", Percent: 100},
			},
		}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			NewRevision:   "svc-00002",
			CanaryPercent: 10,
		}

		result, err := SetCanaryTraffic(context.Background(), client, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.NewRevision != "svc-00002" {
			t.Errorf("NewRevision = %q, want %q", result.NewRevision, "svc-00002")
		}
		if result.NewPercent != 10 {
			t.Errorf("NewPercent = %d, want 10", result.NewPercent)
		}
		if result.StableRevision != "svc-00001" {
			t.Errorf("StableRevision = %q, want %q", result.StableRevision, "svc-00001")
		}
		if result.StablePercent != 90 {
			t.Errorf("StablePercent = %d, want 90", result.StablePercent)
		}

		if len(client.setTargets) != 2 {
			t.Fatalf("SetTraffic called with %d targets, want 2", len(client.setTargets))
		}
		if client.setTargets[0].Tag != "canary" {
			t.Errorf("canary target tag = %q, want %q", client.setTargets[0].Tag, "canary")
		}
	})

	t.Run("invalid canary percent zero", func(t *testing.T) {
		client := &mockTrafficClient{}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			NewRevision:   "svc-00002",
			CanaryPercent: 0,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for canary percent 0")
		}
	})

	t.Run("invalid canary percent 100", func(t *testing.T) {
		client := &mockTrafficClient{}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			NewRevision:   "svc-00002",
			CanaryPercent: 100,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for canary percent 100")
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		client := &mockTrafficClient{}
		config := CanaryConfig{
			NewRevision:   "svc-00002",
			CanaryPercent: 10,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for empty service name")
		}
	})

	t.Run("empty new revision", func(t *testing.T) {
		client := &mockTrafficClient{}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			CanaryPercent: 10,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for empty new revision")
		}
	})

	t.Run("no current traffic", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{},
		}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			NewRevision:   "svc-00002",
			CanaryPercent: 10,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error for no current traffic")
		}
	})

	t.Run("get traffic error", func(t *testing.T) {
		client := &mockTrafficClient{
			getErr: errors.New("network error"),
		}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			NewRevision:   "svc-00002",
			CanaryPercent: 10,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error when GetTraffic fails")
		}
	})

	t.Run("set traffic error", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{
				{RevisionName: "svc-00001", Percent: 100},
			},
			setErr: errors.New("permission denied"),
		}
		config := CanaryConfig{
			ServiceName:   "projects/p/locations/l/services/svc",
			NewRevision:   "svc-00002",
			CanaryPercent: 10,
		}

		_, err := SetCanaryTraffic(context.Background(), client, config)
		if err == nil {
			t.Fatal("expected error when SetTraffic fails")
		}
	})
}

func TestPromoteCanary(t *testing.T) {
	t.Run("promotes tagged canary to 100%", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{
				{RevisionName: "svc-00002", Percent: 10, Tag: "canary"},
				{RevisionName: "svc-00001", Percent: 90},
			},
		}

		err := PromoteCanary(context.Background(), client, "projects/p/locations/l/services/svc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(client.setTargets) != 1 {
			t.Fatalf("SetTraffic called with %d targets, want 1", len(client.setTargets))
		}
		if client.setTargets[0].RevisionName != "svc-00002" {
			t.Errorf("promoted revision = %q, want %q", client.setTargets[0].RevisionName, "svc-00002")
		}
		if client.setTargets[0].Percent != 100 {
			t.Errorf("promoted percent = %d, want 100", client.setTargets[0].Percent)
		}
	})

	t.Run("promotes non-100% revision when no tag", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{
				{RevisionName: "svc-00001", Percent: 90},
				{RevisionName: "svc-00002", Percent: 10},
			},
		}

		err := PromoteCanary(context.Background(), client, "projects/p/locations/l/services/svc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(client.setTargets) != 1 {
			t.Fatalf("SetTraffic called with %d targets, want 1", len(client.setTargets))
		}
		// Should pick svc-00001 since it's <100% and >0% — but actually both are <100%.
		// The function picks the first non-100% revision found.
		if client.setTargets[0].RevisionName != "svc-00001" {
			t.Errorf("promoted revision = %q, want %q", client.setTargets[0].RevisionName, "svc-00001")
		}
	})

	t.Run("no canary found", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{
				{RevisionName: "svc-00001", Percent: 100},
			},
		}

		err := PromoteCanary(context.Background(), client, "projects/p/locations/l/services/svc")
		if err == nil {
			t.Fatal("expected error when no canary revision found")
		}
	})

	t.Run("no current traffic", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{},
		}

		err := PromoteCanary(context.Background(), client, "projects/p/locations/l/services/svc")
		if err == nil {
			t.Fatal("expected error for no current traffic")
		}
	})

	t.Run("empty service name", func(t *testing.T) {
		client := &mockTrafficClient{}

		err := PromoteCanary(context.Background(), client, "")
		if err == nil {
			t.Fatal("expected error for empty service name")
		}
	})

	t.Run("get traffic error", func(t *testing.T) {
		client := &mockTrafficClient{
			getErr: errors.New("network error"),
		}

		err := PromoteCanary(context.Background(), client, "projects/p/locations/l/services/svc")
		if err == nil {
			t.Fatal("expected error when GetTraffic fails")
		}
	})

	t.Run("set traffic error", func(t *testing.T) {
		client := &mockTrafficClient{
			targets: []TrafficTarget{
				{RevisionName: "svc-00002", Percent: 10, Tag: "canary"},
				{RevisionName: "svc-00001", Percent: 90},
			},
			setErr: errors.New("permission denied"),
		}

		err := PromoteCanary(context.Background(), client, "projects/p/locations/l/services/svc")
		if err == nil {
			t.Fatal("expected error when SetTraffic fails")
		}
	})
}
