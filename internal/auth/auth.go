// Package auth handles authentication for the Mint CLI,
// including credential storage and retrieval.
package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Credentials holds a stored API token.
type Credentials struct {
	APIKey string `json:"api_key"`
}

// credentialsDir returns the path to ~/.mint.
func credentialsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mint"), nil
}

// credentialsPath returns the path to ~/.mint/credentials.
func credentialsPath() (string, error) {
	dir, err := credentialsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}

// SaveCredentials writes the API key to ~/.mint/credentials.
func SaveCredentials(apiKey string) error {
	dir, err := credentialsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	creds := Credentials{APIKey: apiKey}
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadToken returns the API token, checking MINT_API_KEY env var first,
// then falling back to ~/.mint/credentials.
func LoadToken() (string, error) {
	if key := os.Getenv("MINT_API_KEY"); key != "" {
		return key, nil
	}
	path, err := credentialsPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", errors.New("not logged in: run 'mint login' or set MINT_API_KEY")
		}
		return "", err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", err
	}
	if creds.APIKey == "" {
		return "", errors.New("credentials file is empty: run 'mint login' or set MINT_API_KEY")
	}
	return creds.APIKey, nil
}
