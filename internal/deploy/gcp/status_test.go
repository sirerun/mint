package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

type mockStatusClient struct {
	service   *ServiceStatus
	revisions []RevisionStatus
	getErr    error
	listErr   error
}

func (m *mockStatusClient) GetService(_ context.Context, _ string) (*ServiceStatus, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.service, nil
}

func (m *mockStatusClient) ListRevisions(_ context.Context, _ string) ([]RevisionStatus, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.revisions, nil
}

func TestGetStatus_Success(t *testing.T) {
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	client := &mockStatusClient{
		service: &ServiceStatus{
			Name:       "projects/my-project/locations/us-central1/services/my-svc",
			URL:        "https://my-svc-abc123.run.app",
			Labels:     map[string]string{"env": "prod", "team": "platform"},
			CreateTime: now.Add(-24 * time.Hour),
			UpdateTime: now,
		},
		revisions: []RevisionStatus{
			{
				Name:           "my-svc-00002",
				CreateTime:     now,
				TrafficPercent: 80,
				Active:         true,
			},
			{
				Name:           "my-svc-00001",
				CreateTime:     now.Add(-24 * time.Hour),
				TrafficPercent: 20,
				Active:         true,
			},
		},
	}

	result, err := GetStatus(context.Background(), client, "my-project", "us-central1", "my-svc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ServiceName != "my-svc" {
		t.Errorf("ServiceName = %q, want %q", result.ServiceName, "my-svc")
	}
	if result.URL != "https://my-svc-abc123.run.app" {
		t.Errorf("URL = %q, want %q", result.URL, "https://my-svc-abc123.run.app")
	}
	if len(result.Labels) != 2 {
		t.Errorf("Labels count = %d, want 2", len(result.Labels))
	}
	if result.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %q, want %q", result.Labels["env"], "prod")
	}
	if len(result.Revisions) != 2 {
		t.Fatalf("Revisions count = %d, want 2", len(result.Revisions))
	}

	rev := result.Revisions[0]
	if rev.Name != "my-svc-00002" {
		t.Errorf("Revision[0].Name = %q, want %q", rev.Name, "my-svc-00002")
	}
	if rev.TrafficPercent != 80 {
		t.Errorf("Revision[0].TrafficPercent = %d, want 80", rev.TrafficPercent)
	}
	if !rev.Active {
		t.Error("Revision[0].Active = false, want true")
	}
	if rev.CreateTime != now.Format(time.RFC3339) {
		t.Errorf("Revision[0].CreateTime = %q, want %q", rev.CreateTime, now.Format(time.RFC3339))
	}
}

func TestGetStatus_ServiceNotFound(t *testing.T) {
	client := &mockStatusClient{
		getErr: fmt.Errorf("service not found"),
	}

	_, err := GetStatus(context.Background(), client, "my-project", "us-central1", "missing-svc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "getting service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "getting service")
	}
	if !strings.Contains(err.Error(), "service not found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "service not found")
	}
}

func TestGetStatus_ListRevisionsError(t *testing.T) {
	client := &mockStatusClient{
		service: &ServiceStatus{
			Name: "projects/p/locations/r/services/s",
			URL:  "https://s.run.app",
		},
		listErr: fmt.Errorf("permission denied"),
	}

	_, err := GetStatus(context.Background(), client, "p", "r", "s")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "listing revisions") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "listing revisions")
	}
}

func TestFormatStatus_JSON(t *testing.T) {
	result := &StatusResult{
		ServiceName: "test-svc",
		URL:         "https://test-svc.run.app",
		Labels:      map[string]string{"env": "test"},
		Revisions: []RevisionInfo{
			{
				Name:           "test-svc-00001",
				TrafficPercent: 100,
				Active:         true,
				CreateTime:     "2026-03-08T12:00:00Z",
			},
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
	if parsed.URL != "https://test-svc.run.app" {
		t.Errorf("parsed URL = %q, want %q", parsed.URL, "https://test-svc.run.app")
	}
	if len(parsed.Revisions) != 1 {
		t.Fatalf("parsed Revisions count = %d, want 1", len(parsed.Revisions))
	}
	if parsed.Revisions[0].TrafficPercent != 100 {
		t.Errorf("parsed Revisions[0].TrafficPercent = %d, want 100", parsed.Revisions[0].TrafficPercent)
	}
}

func TestFormatStatus_Text(t *testing.T) {
	result := &StatusResult{
		ServiceName: "my-svc",
		URL:         "https://my-svc.run.app",
		Labels:      map[string]string{"env": "prod"},
		Revisions: []RevisionInfo{
			{
				Name:           "my-svc-00002",
				TrafficPercent: 75,
				Active:         true,
				CreateTime:     "2026-03-08T12:00:00Z",
			},
			{
				Name:           "my-svc-00001",
				TrafficPercent: 25,
				Active:         false,
				CreateTime:     "2026-03-07T12:00:00Z",
			},
		},
	}

	output := FormatStatus(result, false)

	checks := []string{
		"my-svc",
		"https://my-svc.run.app",
		"env=prod",
		"my-svc-00002",
		"75%",
		"yes",
		"my-svc-00001",
		"25%",
		"no",
		"NAME",
		"TRAFFIC",
		"ACTIVE",
		"CREATED",
	}

	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("text output missing %q\nOutput:\n%s", want, output)
		}
	}
}
