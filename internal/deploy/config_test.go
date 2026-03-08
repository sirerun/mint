package deploy

import (
	"os"
	"testing"
)

func TestDeployConfigValidate(t *testing.T) {
	// Create a temporary directory to use as a valid source dir.
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		config  DeployConfig
		wantErr string
	}{
		{
			name: "valid config",
			config: DeployConfig{
				ProjectID:    "my-project",
				Region:       "us-central1",
				SourceDir:    tmpDir,
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "",
		},
		{
			name: "missing project ID",
			config: DeployConfig{
				SourceDir:    tmpDir,
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "project ID is required",
		},
		{
			name: "missing source dir",
			config: DeployConfig{
				ProjectID:    "my-project",
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "source directory is required",
		},
		{
			name: "source dir does not exist",
			config: DeployConfig{
				ProjectID:    "my-project",
				SourceDir:    "/nonexistent/path/that/does/not/exist",
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "does not exist",
		},
		{
			name: "source dir is a file",
			config: DeployConfig{
				ProjectID:    "my-project",
				SourceDir:    "", // set below
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "is not a directory",
		},
		{
			name: "canary too high",
			config: DeployConfig{
				ProjectID:    "my-project",
				SourceDir:    tmpDir,
				Canary:       100,
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "canary percentage must be between 0 and 99",
		},
		{
			name: "canary negative",
			config: DeployConfig{
				ProjectID:    "my-project",
				SourceDir:    tmpDir,
				Canary:       -1,
				Timeout:      300,
				MaxInstances: 10,
			},
			wantErr: "canary percentage must be between 0 and 99",
		},
		{
			name: "timeout zero",
			config: DeployConfig{
				ProjectID:    "my-project",
				SourceDir:    tmpDir,
				Timeout:      0,
				MaxInstances: 10,
			},
			wantErr: "timeout must be greater than 0",
		},
		{
			name: "max instances zero",
			config: DeployConfig{
				ProjectID:    "my-project",
				SourceDir:    tmpDir,
				Timeout:      300,
				MaxInstances: 0,
			},
			wantErr: "max-instances must be greater than 0",
		},
	}

	// Create a temporary file for the "source dir is a file" test case.
	tmpFile, err := os.CreateTemp(t.TempDir(), "notadir")
	if err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Patch the "source dir is a file" test case.
	for i := range tests {
		if tests[i].name == "source dir is a file" {
			tests[i].config.SourceDir = tmpFile.Name()
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() expected error containing %q, got nil", tt.wantErr)
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("Validate() error = %q, want substring %q", got, tt.wantErr)
			}
		})
	}
}

func TestParseSecretFlag(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEnv    string
		wantSecret string
		wantErr    string
	}{
		{
			name:       "valid mapping",
			input:      "DB_PASSWORD=my-db-secret",
			wantEnv:    "DB_PASSWORD",
			wantSecret: "my-db-secret",
		},
		{
			name:       "valid with equals in secret name",
			input:      "API_KEY=projects/p/secrets/s/versions/latest",
			wantEnv:    "API_KEY",
			wantSecret: "projects/p/secrets/s/versions/latest",
		},
		{
			name:    "missing equals",
			input:   "NOSEPARATOR",
			wantErr: "expected format ENV_VAR=secret-name",
		},
		{
			name:    "empty env var",
			input:   "=secret-name",
			wantErr: "environment variable name is empty",
		},
		{
			name:    "empty secret name",
			input:   "ENV_VAR=",
			wantErr: "secret name is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSecretFlag(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseSecretFlag(%q) expected error containing %q, got nil", tt.input, tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Errorf("ParseSecretFlag(%q) error = %q, want substring %q", tt.input, err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSecretFlag(%q) unexpected error: %v", tt.input, err)
			}
			if got.EnvVar != tt.wantEnv {
				t.Errorf("ParseSecretFlag(%q).EnvVar = %q, want %q", tt.input, got.EnvVar, tt.wantEnv)
			}
			if got.SecretName != tt.wantSecret {
				t.Errorf("ParseSecretFlag(%q).SecretName = %q, want %q", tt.input, got.SecretName, tt.wantSecret)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
