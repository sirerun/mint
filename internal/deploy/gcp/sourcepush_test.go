package gcp

import (
	"context"
	"errors"
	"testing"
)

// mockGitClient records calls and returns configured errors.
type mockGitClient struct {
	initCalled      bool
	addAllCalled    bool
	commitCalled    bool
	addRemoteCalled bool
	pushCalled      bool
	hasRemoteCalled bool

	commitMessage string
	remoteName    string
	remoteURL     string

	hasRemoteResult bool
	initErr         error
	addAllErr       error
	commitErr       error
	addRemoteErr    error
	pushErr         error
	hasRemoteErr    error
}

func (m *mockGitClient) Init(_ string) error {
	m.initCalled = true
	return m.initErr
}

func (m *mockGitClient) AddAll(_ string) error {
	m.addAllCalled = true
	return m.addAllErr
}

func (m *mockGitClient) Commit(_ string, message string) error {
	m.commitCalled = true
	m.commitMessage = message
	return m.commitErr
}

func (m *mockGitClient) AddRemote(_ string, name, url string) error {
	m.addRemoteCalled = true
	m.remoteName = name
	m.remoteURL = url
	return m.addRemoteErr
}

func (m *mockGitClient) Push(_ string, _, _ string) error {
	m.pushCalled = true
	return m.pushErr
}

func (m *mockGitClient) HasRemote(_ string, _ string) (bool, error) {
	m.hasRemoteCalled = true
	return m.hasRemoteResult, m.hasRemoteErr
}

func TestRepoURL(t *testing.T) {
	tests := []struct {
		projectID string
		repoName  string
		want      string
	}{
		{
			projectID: "my-project",
			repoName:  "mint-mcp-petstore",
			want:      "https://source.developers.google.com/p/my-project/r/mint-mcp-petstore",
		},
		{
			projectID: "proj-123",
			repoName:  "my-repo",
			want:      "https://source.developers.google.com/p/proj-123/r/my-repo",
		},
	}

	for _, tt := range tests {
		got := RepoURL(tt.projectID, tt.repoName)
		if got != tt.want {
			t.Errorf("RepoURL(%q, %q) = %q, want %q", tt.projectID, tt.repoName, got, tt.want)
		}
	}
}

func TestPushSource_Success(t *testing.T) {
	mock := &mockGitClient{}
	config := SourcePushConfig{
		SourceDir: "/tmp/gen",
		ProjectID: "my-project",
		RepoName:  "mint-mcp-petstore",
		SpecHash:  "abc123",
	}

	result, err := PushSource(context.Background(), mock, config)
	if err != nil {
		t.Fatalf("PushSource() unexpected error: %v", err)
	}

	if !mock.initCalled {
		t.Error("expected Init to be called")
	}
	if !mock.hasRemoteCalled {
		t.Error("expected HasRemote to be called")
	}
	if !mock.addRemoteCalled {
		t.Error("expected AddRemote to be called")
	}
	if !mock.addAllCalled {
		t.Error("expected AddAll to be called")
	}
	if !mock.commitCalled {
		t.Error("expected Commit to be called")
	}
	if !mock.pushCalled {
		t.Error("expected Push to be called")
	}

	wantURL := "https://source.developers.google.com/p/my-project/r/mint-mcp-petstore"
	if result.RepoURL != wantURL {
		t.Errorf("RepoURL = %q, want %q", result.RepoURL, wantURL)
	}

	if mock.commitMessage == "" {
		t.Error("expected commit message to be set")
	}
	if mock.remoteName != "google" {
		t.Errorf("remote name = %q, want %q", mock.remoteName, "google")
	}
	if mock.remoteURL != wantURL {
		t.Errorf("remote URL = %q, want %q", mock.remoteURL, wantURL)
	}
}

func TestPushSource_AlreadyHasRemote(t *testing.T) {
	mock := &mockGitClient{
		hasRemoteResult: true,
	}
	config := SourcePushConfig{
		SourceDir: "/tmp/gen",
		ProjectID: "my-project",
		RepoName:  "mint-mcp-petstore",
		SpecHash:  "abc123",
	}

	_, err := PushSource(context.Background(), mock, config)
	if err != nil {
		t.Fatalf("PushSource() unexpected error: %v", err)
	}

	if mock.addRemoteCalled {
		t.Error("expected AddRemote to NOT be called when remote already exists")
	}
	if !mock.pushCalled {
		t.Error("expected Push to still be called")
	}
}

func TestPushSource_InitFails(t *testing.T) {
	mock := &mockGitClient{
		initErr: errors.New("init failed"),
	}
	config := SourcePushConfig{
		SourceDir: "/tmp/gen",
		ProjectID: "my-project",
		RepoName:  "mint-mcp-petstore",
		SpecHash:  "abc123",
	}

	_, err := PushSource(context.Background(), mock, config)
	if err == nil {
		t.Fatal("expected error when Init fails")
	}
	if !errors.Is(err, mock.initErr) {
		t.Errorf("error = %v, want wrapped %v", err, mock.initErr)
	}
	if mock.addAllCalled {
		t.Error("expected AddAll to NOT be called after Init failure")
	}
}

func TestPushSource_PushFails(t *testing.T) {
	mock := &mockGitClient{
		pushErr: errors.New("push failed"),
	}
	config := SourcePushConfig{
		SourceDir: "/tmp/gen",
		ProjectID: "my-project",
		RepoName:  "mint-mcp-petstore",
		SpecHash:  "abc123",
	}

	_, err := PushSource(context.Background(), mock, config)
	if err == nil {
		t.Fatal("expected error when Push fails")
	}
	if !errors.Is(err, mock.pushErr) {
		t.Errorf("error = %v, want wrapped %v", err, mock.pushErr)
	}
}
