package managed

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatStatusHuman(t *testing.T) {
	status := &ServerStatus{
		ServiceID: "svc-123",
		URL:       "https://my-server.mintmcp.com",
		State:     "running",
		Revisions: []RevisionInfo{
			{Name: "rev-1", State: "active", TrafficPercent: 80},
			{Name: "rev-2", State: "deploying", TrafficPercent: 20},
		},
		CreatedAt: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
	}

	out := FormatStatus(status, false)

	checks := []string{
		"svc-123",
		"https://my-server.mintmcp.com",
		"running",
		"2026-03-13 10:00:00 UTC",
		"REVISION",
		"rev-1",
		"active",
		"80%",
		"rev-2",
		"deploying",
		"20%",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestFormatStatusJSON(t *testing.T) {
	status := &ServerStatus{
		ServiceID: "svc-123",
		URL:       "https://my-server.mintmcp.com",
		State:     "running",
		Revisions: []RevisionInfo{
			{Name: "rev-1", State: "active", TrafficPercent: 100},
		},
		CreatedAt: time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC),
	}

	out := FormatStatus(status, true)

	var parsed ServerStatus
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if parsed.ServiceID != "svc-123" {
		t.Errorf("ServiceID = %q, want %q", parsed.ServiceID, "svc-123")
	}
	if parsed.State != "running" {
		t.Errorf("State = %q, want %q", parsed.State, "running")
	}
	if len(parsed.Revisions) != 1 {
		t.Fatalf("Revisions length = %d, want 1", len(parsed.Revisions))
	}
}

func TestFormatStatusNoRevisions(t *testing.T) {
	status := &ServerStatus{
		ServiceID: "svc-456",
		URL:       "https://other.mintmcp.com",
		State:     "building",
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	out := FormatStatus(status, false)

	if !strings.Contains(out, "svc-456") {
		t.Error("output missing service ID")
	}
	if strings.Contains(out, "REVISION") {
		t.Error("output should not contain revision header when there are no revisions")
	}
}

func TestFormatServerListHuman(t *testing.T) {
	servers := []ServerSummary{
		{ServiceID: "svc-1", ServiceName: "server-a", URL: "https://a.mintmcp.com", State: "running"},
		{ServiceID: "svc-2", ServiceName: "server-b", URL: "https://b.mintmcp.com", State: "stopped"},
	}

	out := FormatServerList(servers, false)

	checks := []string{
		"SERVICE ID",
		"NAME",
		"URL",
		"STATE",
		"svc-1",
		"server-a",
		"https://a.mintmcp.com",
		"running",
		"svc-2",
		"server-b",
		"stopped",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestFormatServerListJSON(t *testing.T) {
	servers := []ServerSummary{
		{ServiceID: "svc-1", ServiceName: "server-a", URL: "https://a.mintmcp.com", State: "running"},
	}

	out := FormatServerList(servers, true)

	var parsed []ServerSummary
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if len(parsed) != 1 {
		t.Fatalf("length = %d, want 1", len(parsed))
	}
	if parsed[0].ServiceID != "svc-1" {
		t.Errorf("ServiceID = %q, want %q", parsed[0].ServiceID, "svc-1")
	}
}

func TestFormatServerListEmpty(t *testing.T) {
	out := FormatServerList(nil, false)
	if !strings.Contains(out, "No servers found") {
		t.Errorf("output = %q, want 'No servers found'", out)
	}
}

func TestFormatServerListEmptyJSON(t *testing.T) {
	out := FormatServerList([]ServerSummary{}, true)

	var parsed []ServerSummary
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}
	if len(parsed) != 0 {
		t.Fatalf("length = %d, want 0", len(parsed))
	}
}
