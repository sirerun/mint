package gcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExecGitClientInterfaceCheck(t *testing.T) {
	var _ GitClient = (*ExecGitClient)(nil)
}

func TestNewExecGitClient(t *testing.T) {
	client, err := NewExecGitClient()
	if err != nil {
		t.Skipf("git not in PATH: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestExecGitClientInitAndAddAll(t *testing.T) {
	client, err := NewExecGitClient()
	if err != nil {
		t.Skipf("git not in PATH: %v", err)
	}

	dir := t.TempDir()

	if err := client.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Verify .git directory exists.
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected .git directory: %v", err)
	}

	// Create a file and stage it.
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := client.AddAll(dir); err != nil {
		t.Fatalf("AddAll: %v", err)
	}
}

func TestExecGitClientHasRemote(t *testing.T) {
	client, err := NewExecGitClient()
	if err != nil {
		t.Skipf("git not in PATH: %v", err)
	}

	dir := t.TempDir()
	if err := client.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	has, err := client.HasRemote(dir, "origin")
	if err != nil {
		t.Fatalf("HasRemote: %v", err)
	}
	if has {
		t.Fatal("expected no remote")
	}

	if err := client.AddRemote(dir, "origin", "https://example.com/repo.git"); err != nil {
		t.Fatalf("AddRemote: %v", err)
	}

	has, err = client.HasRemote(dir, "origin")
	if err != nil {
		t.Fatalf("HasRemote: %v", err)
	}
	if !has {
		t.Fatal("expected remote to exist")
	}
}

func TestExecGitClientCommit(t *testing.T) {
	client, err := NewExecGitClient()
	if err != nil {
		t.Skipf("git not in PATH: %v", err)
	}

	dir := t.TempDir()
	if err := client.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Configure user for commit.
	if err := client.run(dir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("config email: %v", err)
	}
	if err := client.run(dir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("config name: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := client.AddAll(dir); err != nil {
		t.Fatalf("AddAll: %v", err)
	}
	if err := client.Commit(dir, "initial commit"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}
