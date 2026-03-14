package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeployFromSourceSuccess(t *testing.T) {
	var callOrder []string

	pollCount := int32(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/sources":
			callOrder = append(callOrder, "upload")
			writeBody(w, http.StatusOK, "src-abc123")

		case r.Method == http.MethodPost && r.URL.Path == "/services":
			callOrder = append(callOrder, "deploy")
			var input DeployInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Errorf("decoding deploy input: %v", err)
			}
			if input.Source != "src-abc123" {
				t.Errorf("source = %q, want %q", input.Source, "src-abc123")
			}
			if input.ServiceName != "my-server" {
				t.Errorf("service_name = %q, want %q", input.ServiceName, "my-server")
			}
			mustEncode(w, DeployOutput{
				URL:       "https://my-server.mintmcp.com",
				ServiceID: "svc-123",
				BuildID:   "build-456",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/services/svc-123":
			callOrder = append(callOrder, "poll")
			count := atomic.AddInt32(&pollCount, 1)
			status := ServerStatus{ServiceID: "svc-123", State: "building"}
			if count >= 2 {
				status.State = "ready"
				status.URL = "https://my-server.mintmcp.com"
			}
			mustEncode(w, status)

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	out, err := DeployFromSource(context.Background(), client, sourceDir, "my-server", true, &stderr)
	if err != nil {
		t.Fatalf("DeployFromSource: %v", err)
	}

	if out.URL != "https://my-server.mintmcp.com" {
		t.Errorf("URL = %q, want %q", out.URL, "https://my-server.mintmcp.com")
	}
	if out.ServiceID != "svc-123" {
		t.Errorf("ServiceID = %q, want %q", out.ServiceID, "svc-123")
	}

	if len(callOrder) < 3 {
		t.Fatalf("expected at least 3 calls, got %d: %v", len(callOrder), callOrder)
	}
	if callOrder[0] != "upload" {
		t.Errorf("first call = %q, want %q", callOrder[0], "upload")
	}
	if callOrder[1] != "deploy" {
		t.Errorf("second call = %q, want %q", callOrder[1], "deploy")
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "creating source tarball") {
		t.Error("stderr missing 'creating source tarball'")
	}
	if !strings.Contains(stderrStr, "deploying") {
		t.Error("stderr missing 'deploying'")
	}
}

func TestDeployFromSourceUploadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sources" {
			writeBody(w, http.StatusInternalServerError, "upload failed")
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	_, err := DeployFromSource(context.Background(), client, sourceDir, "my-server", true, &stderr)
	if err == nil {
		t.Fatal("expected error when upload fails")
	}
	if !strings.Contains(err.Error(), "uploading source") {
		t.Errorf("error = %q, want to contain 'uploading source'", err.Error())
	}
}

func TestDeployFromSourceDeployError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sources":
			writeBody(w, http.StatusOK, "src-123")
		case r.URL.Path == "/services" && r.Method == http.MethodPost:
			writeBody(w, http.StatusBadRequest, "bad request")
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	_, err := DeployFromSource(context.Background(), client, sourceDir, "my-server", true, &stderr)
	if err == nil {
		t.Fatal("expected error when deploy fails")
	}
	if !strings.Contains(err.Error(), "deploying") {
		t.Errorf("error = %q, want to contain 'deploying'", err.Error())
	}
}

func TestDeployFromSourceBuildFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sources":
			writeBody(w, http.StatusOK, "src-123")
		case r.URL.Path == "/services" && r.Method == http.MethodPost:
			mustEncode(w, DeployOutput{
				ServiceID: "svc-fail",
				BuildID:   "build-fail",
			})
		case r.URL.Path == "/services/svc-fail" && r.Method == http.MethodGet:
			mustEncode(w, ServerStatus{
				ServiceID: "svc-fail",
				State:     "failed",
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	_, err := DeployFromSource(context.Background(), client, sourceDir, "my-server", true, &stderr)
	if err == nil {
		t.Fatal("expected error when build fails")
	}
	if !strings.Contains(err.Error(), "deployment failed") {
		t.Errorf("error = %q, want to contain 'deployment failed'", err.Error())
	}
}

func TestDeployFromSourcePollTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/sources":
			writeBody(w, http.StatusOK, "src-123")
		case r.URL.Path == "/services" && r.Method == http.MethodPost:
			mustEncode(w, DeployOutput{
				ServiceID: "svc-slow",
				BuildID:   "build-slow",
			})
		case r.URL.Path == "/services/svc-slow" && r.Method == http.MethodGet:
			mustEncode(w, ServerStatus{
				ServiceID: "svc-slow",
				State:     "building",
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	_, err := DeployFromSource(ctx, client, sourceDir, "my-server", true, &stderr)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want to contain 'timed out'", err.Error())
	}
}

func TestDeployFromSourceTarballError(t *testing.T) {
	client := NewClient("http://localhost:0", "test-token")
	var stderr bytes.Buffer
	_, err := DeployFromSource(context.Background(), client, "/nonexistent/path", "my-server", true, &stderr)
	if err == nil {
		t.Fatal("expected error for nonexistent source directory")
	}
	if !strings.Contains(err.Error(), "creating tarball") {
		t.Errorf("error = %q, want to contain 'creating tarball'", err.Error())
	}
}

func TestPollDeploymentStateTransitions(t *testing.T) {
	states := []string{"building", "deploying", "ready"}
	idx := int32(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		i := atomic.LoadInt32(&idx)
		if int(i) >= len(states) {
			i = int32(len(states) - 1)
		}
		mustEncode(w, ServerStatus{
			ServiceID: "svc-1",
			State:     states[i],
		})
		atomic.AddInt32(&idx, 1)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	err := pollDeployment(context.Background(), client, "svc-1", &stderr)
	if err != nil {
		t.Fatalf("pollDeployment: %v", err)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "building") {
		t.Error("stderr missing 'building' state")
	}
	if !strings.Contains(stderrStr, "deploying") {
		t.Error("stderr missing 'deploying' state")
	}
	if !strings.Contains(stderrStr, "ready") {
		t.Error("stderr missing 'ready' state")
	}
}

func TestPollDeploymentErrorState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mustEncode(w, ServerStatus{
			ServiceID: "svc-1",
			State:     "error",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	err := pollDeployment(context.Background(), client, "svc-1", &stderr)
	if err == nil {
		t.Fatal("expected error for error state")
	}
	if !strings.Contains(err.Error(), "deployment failed") {
		t.Errorf("error = %q, want to contain 'deployment failed'", err.Error())
	}
}

func TestPollDeploymentHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeBody(w, http.StatusInternalServerError, "server error")
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	err := pollDeployment(context.Background(), client, "svc-1", &stderr)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "polling status") {
		t.Errorf("error = %q, want to contain 'polling status'", err.Error())
	}
}

func TestPollDeploymentRunningIsTerminal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mustEncode(w, ServerStatus{
			ServiceID: "svc-1",
			State:     "running",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test-token")
	var stderr bytes.Buffer
	err := pollDeployment(context.Background(), client, "svc-1", &stderr)
	if err != nil {
		t.Fatalf("pollDeployment: %v", err)
	}
}
