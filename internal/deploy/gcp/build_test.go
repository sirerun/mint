package gcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockBuildClient implements BuildClient for testing.
type mockBuildClient struct {
	result *BuildResult
	err    error
}

func (m *mockBuildClient) CreateBuild(_ context.Context, _ string, build *BuildConfig) (*BuildResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.result != nil {
		return m.result, nil
	}
	return &BuildResult{
		ImageURI: build.ImageURI,
		LogURL:   "https://console.cloud.google.com/cloud-build/builds/abc123",
		Duration: 45 * time.Second,
		Status:   "SUCCESS",
	}, nil
}

func TestBuildImage_Success(t *testing.T) {
	client := &mockBuildClient{}
	config := BuildConfig{
		SourceDir: t.TempDir(),
		ImageURI:  "us-central1-docker.pkg.dev/my-proj/repo/svc:latest",
		ProjectID: "my-proj",
	}

	result, err := BuildImage(context.Background(), client, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ImageURI != config.ImageURI {
		t.Errorf("ImageURI = %q, want %q", result.ImageURI, config.ImageURI)
	}
	if result.Status != "SUCCESS" {
		t.Errorf("Status = %q, want %q", result.Status, "SUCCESS")
	}
}

func TestBuildImage_ClientError(t *testing.T) {
	client := &mockBuildClient{err: errors.New("quota exceeded")}
	config := BuildConfig{
		SourceDir: t.TempDir(),
		ImageURI:  "us-central1-docker.pkg.dev/my-proj/repo/svc:latest",
		ProjectID: "my-proj",
	}

	_, err := BuildImage(context.Background(), client, config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "build failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "build failed")
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "quota exceeded")
	}
}

func TestBuildImage_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config BuildConfig
		want   string
	}{
		{
			name:   "missing source dir",
			config: BuildConfig{SourceDir: "", ImageURI: "img:tag", ProjectID: "proj"},
			want:   "source directory is required",
		},
		{
			name:   "nonexistent source dir",
			config: BuildConfig{SourceDir: "/nonexistent/path/xyz", ImageURI: "img:tag", ProjectID: "proj"},
			want:   "source directory",
		},
		{
			name:   "missing image URI",
			config: BuildConfig{ImageURI: "", ProjectID: "proj"},
			want:   "source directory is required",
		},
		{
			name:   "missing project ID",
			config: BuildConfig{ImageURI: "img:tag", ProjectID: ""},
			want:   "source directory is required",
		},
	}

	client := &mockBuildClient{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildImage(context.Background(), client, tt.config)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestImageURI(t *testing.T) {
	got := ImageURI("us-central1", "my-project", "my-repo", "my-service", "v1.0.0")
	want := "us-central1-docker.pkg.dev/my-project/my-repo/my-service:v1.0.0"
	if got != want {
		t.Errorf("ImageURI() = %q, want %q", got, want)
	}
}
