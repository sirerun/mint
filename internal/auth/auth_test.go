package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	// Use a temp dir as HOME so we don't write to real ~/.mint.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Ensure MINT_API_KEY is not set.
	t.Setenv("MINT_API_KEY", "")

	apiKey := "mint_test_key_123"
	if err := SaveCredentials(apiKey); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file permissions.
	path := filepath.Join(tmpHome, ".mint", "credentials")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials permissions = %o, want 600", perm)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got != apiKey {
		t.Errorf("LoadToken = %q, want %q", got, apiKey)
	}
}

func TestLoadTokenEnvOverride(t *testing.T) {
	envKey := "mint_env_key_456"
	t.Setenv("MINT_API_KEY", envKey)

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got != envKey {
		t.Errorf("LoadToken = %q, want %q", got, envKey)
	}
}

func TestLoadTokenNotLoggedIn(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("MINT_API_KEY", "")

	_, err := LoadToken()
	if err == nil {
		t.Fatal("LoadToken: expected error when not logged in")
	}
}

func TestLoadTokenEmptyCredentials(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("MINT_API_KEY", "")

	dir := filepath.Join(tmpHome, ".mint")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Write credentials with empty API key.
	if err := os.WriteFile(filepath.Join(dir, "credentials"), []byte(`{"api_key":""}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToken()
	if err == nil {
		t.Fatal("LoadToken: expected error for empty credentials")
	}
}

func TestLoadTokenInvalidJSON(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("MINT_API_KEY", "")

	dir := filepath.Join(tmpHome, ".mint")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "credentials"), []byte(`{bad json}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToken()
	if err == nil {
		t.Fatal("LoadToken: expected error for invalid JSON")
	}
}

func TestCredentialsDirPermissions(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if err := SaveCredentials("test-key"); err != nil {
		t.Fatal(err)
	}

	// Verify .mint directory permissions.
	dir := filepath.Join(tmpHome, ".mint")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf(".mint dir permissions = %o, want 700", perm)
	}
}
