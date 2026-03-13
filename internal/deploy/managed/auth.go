package managed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const credentialsFile = ".config/mint/credentials"

// LoadToken reads the API token from the SIRE_API_TOKEN environment variable.
// If the environment variable is not set, it falls back to ~/.config/mint/credentials.
// Returns an error with a clear message when no token is found.
func LoadToken() (string, error) {
	if token := os.Getenv("SIRE_API_TOKEN"); token != "" {
		return token, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no API token found. Set SIRE_API_TOKEN or run 'mint login'")
	}

	path := filepath.Join(home, credentialsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("no API token found. Set SIRE_API_TOKEN or run 'mint login'")
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("no API token found. Set SIRE_API_TOKEN or run 'mint login'")
	}

	return token, nil
}

// SaveToken writes the API token to ~/.config/mint/credentials.
func SaveToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}

	dir := filepath.Join(home, filepath.Dir(credentialsFile))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	path := filepath.Join(home, credentialsFile)
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("writing credentials: %w", err)
	}

	return nil
}
