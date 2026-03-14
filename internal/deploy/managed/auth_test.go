package managed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTokenFromEnv(t *testing.T) {
	t.Setenv("MINT_API_TOKEN", "env-token-123")

	token, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if token != "env-token-123" {
		t.Errorf("token = %q, want %q", token, "env-token-123")
	}
}

func TestLoadTokenFromFile(t *testing.T) {
	t.Setenv("MINT_API_TOKEN", "")

	// Override HOME so we read from a temp directory.
	home := t.TempDir()
	t.Setenv("HOME", home)

	credDir := filepath.Join(home, ".config", "mint")
	if err := os.MkdirAll(credDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credDir, "credentials"), []byte("file-token-456\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if token != "file-token-456" {
		t.Errorf("token = %q, want %q", token, "file-token-456")
	}
}

func TestLoadTokenEnvTakesPrecedence(t *testing.T) {
	t.Setenv("MINT_API_TOKEN", "env-wins")

	home := t.TempDir()
	t.Setenv("HOME", home)

	credDir := filepath.Join(home, ".config", "mint")
	if err := os.MkdirAll(credDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credDir, "credentials"), []byte("file-loses\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	token, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if token != "env-wins" {
		t.Errorf("token = %q, want %q", token, "env-wins")
	}
}

func TestLoadTokenMissing(t *testing.T) {
	t.Setenv("MINT_API_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	_, err := LoadToken()
	if err == nil {
		t.Fatal("expected error when no token is available")
	}
	if !strings.Contains(err.Error(), "no API token found") {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "mint login") {
		t.Errorf("error should mention 'mint login': %v", err)
	}
}

func TestLoadTokenEmptyFile(t *testing.T) {
	t.Setenv("MINT_API_TOKEN", "")

	home := t.TempDir()
	t.Setenv("HOME", home)

	credDir := filepath.Join(home, ".config", "mint")
	if err := os.MkdirAll(credDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credDir, "credentials"), []byte("  \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToken()
	if err == nil {
		t.Fatal("expected error for empty credentials file")
	}
}

func TestSaveToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := SaveToken("saved-token-789"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	// Verify file was created with correct content and permissions.
	path := filepath.Join(home, ".config", "mint", "credentials")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading credentials: %v", err)
	}
	if strings.TrimSpace(string(data)) != "saved-token-789" {
		t.Errorf("file content = %q, want %q", string(data), "saved-token-789\n")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestSaveTokenOverwrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := SaveToken("first-token"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}
	if err := SaveToken("second-token"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	path := filepath.Join(home, ".config", "mint", "credentials")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "second-token" {
		t.Errorf("file content = %q, want %q", string(data), "second-token")
	}
}
