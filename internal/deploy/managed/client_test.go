package managed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeploy(t *testing.T) {
	want := DeployOutput{
		URL:       "https://my-server.sire.run",
		ServiceID: "svc-123",
		BuildID:   "build-456",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/services" {
			t.Errorf("expected /services, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}

		var input DeployInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if input.ServiceName != "my-server" {
			t.Errorf("expected service name my-server, got %s", input.ServiceName)
		}
		if !input.Public {
			t.Error("expected public to be true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	got, err := client.Deploy(context.Background(), &DeployInput{
		Source:      "/tmp/source",
		ServiceName: "my-server",
		Public:      true,
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if got.URL != want.URL {
		t.Errorf("URL = %q, want %q", got.URL, want.URL)
	}
	if got.ServiceID != want.ServiceID {
		t.Errorf("ServiceID = %q, want %q", got.ServiceID, want.ServiceID)
	}
	if got.BuildID != want.BuildID {
		t.Errorf("BuildID = %q, want %q", got.BuildID, want.BuildID)
	}
}

func TestDeployHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.Deploy(context.Background(), &DeployInput{ServiceName: "fail"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestStatus(t *testing.T) {
	createdAt := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	want := ServerStatus{
		ServiceID: "svc-123",
		URL:       "https://my-server.sire.run",
		State:     "running",
		Revisions: []RevisionInfo{
			{Name: "rev-1", State: "active", TrafficPercent: 100},
		},
		CreatedAt: createdAt,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/services/svc-123" {
			t.Errorf("expected /services/svc-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	got, err := client.Status(context.Background(), "svc-123")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.ServiceID != want.ServiceID {
		t.Errorf("ServiceID = %q, want %q", got.ServiceID, want.ServiceID)
	}
	if got.State != want.State {
		t.Errorf("State = %q, want %q", got.State, want.State)
	}
	if len(got.Revisions) != 1 {
		t.Fatalf("Revisions length = %d, want 1", len(got.Revisions))
	}
	if got.Revisions[0].TrafficPercent != 100 {
		t.Errorf("TrafficPercent = %d, want 100", got.Revisions[0].TrafficPercent)
	}
}

func TestStatusHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.Status(context.Background(), "svc-missing")
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/services/svc-123" {
			t.Errorf("expected /services/svc-123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	if err := client.Delete(context.Background(), "svc-123"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDeleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	if err := client.Delete(context.Background(), "svc-123"); err == nil {
		t.Fatal("expected error for HTTP 403")
	}
}

func TestListServers(t *testing.T) {
	want := []ServerSummary{
		{ServiceID: "svc-1", ServiceName: "server-a", URL: "https://a.sire.run", State: "running"},
		{ServiceID: "svc-2", ServiceName: "server-b", URL: "https://b.sire.run", State: "stopped"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/services" {
			t.Errorf("expected /services, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	got, err := client.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListServers length = %d, want 2", len(got))
	}
	if got[0].ServiceName != "server-a" {
		t.Errorf("got[0].ServiceName = %q, want %q", got[0].ServiceName, "server-a")
	}
	if got[1].State != "stopped" {
		t.Errorf("got[1].State = %q, want %q", got[1].State, "stopped")
	}
}

func TestListServersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	got, err := client.ListServers(context.Background())
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListServers length = %d, want 0", len(got))
	}
}

func TestNewClientDefaultBaseURL(t *testing.T) {
	client := NewClient("", "tok")
	hc := client.(*httpClient)
	if hc.baseURL != "https://api.sire.run/v1/hosting" {
		t.Errorf("baseURL = %q, want default", hc.baseURL)
	}
}

func TestDeployInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.Deploy(context.Background(), &DeployInput{ServiceName: "bad"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestStatusInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("{{bad"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.Status(context.Background(), "svc-1")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestListServersInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.ListServers(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestListServersHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unavailable"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.ListServers(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 503")
	}
}

func TestDoJSONConnectionRefused(t *testing.T) {
	// Use a server that's immediately closed to trigger connection error.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	client := NewClient(srv.URL, "test-token")
	_, err := client.Status(context.Background(), "svc-1")
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}
