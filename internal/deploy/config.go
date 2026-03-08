package deploy

import (
	"fmt"
	"os"
	"strings"
)

// DeployConfig holds all configuration for a deployment.
type DeployConfig struct {
	// Required
	ProjectID string
	Region    string // default "us-central1"
	SourceDir string // path to generated server directory

	// Optional
	ServiceName  string          // default derived from source dir name
	ImageTag     string          // default "latest"
	Public       bool            // allow unauthenticated access
	Canary       int             // traffic percentage for canary (0 = full rollout)
	VPC          string          // VPC connector name
	WAF          bool            // enable Cloud Armor
	Internal     bool            // internal-only ingress
	KMSKey       string          // CMEK encryption key
	Timeout      int             // request timeout in seconds (default 300)
	MaxInstances int             // default 10
	MinInstances int             // default 0
	Secrets      []SecretMapping // ENV_VAR=secret-name pairs
	CI           bool            // generate CI workflow
	Promote      bool            // promote canary to 100%
	CPUAlways    bool            // allocate CPU when idle (for SSE)
	DebugImage   bool            // use alpine base for debugging
	NoSourceRepo bool            // skip Cloud Source Repositories push
}

// SecretMapping maps an environment variable to a Secret Manager secret.
type SecretMapping struct {
	EnvVar     string
	SecretName string
}

// DeployResult holds the result of a deployment.
type DeployResult struct {
	ServiceURL   string
	RevisionName string
	Status       string
	ProjectID    string
	Region       string
	ServiceName  string
}

// Validate checks that required fields are set and values are valid.
func (c *DeployConfig) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("project ID is required: use --project flag or set GOOGLE_CLOUD_PROJECT")
	}
	if c.SourceDir == "" {
		return fmt.Errorf("source directory is required: use --source flag")
	}
	info, err := os.Stat(c.SourceDir)
	if err != nil {
		return fmt.Errorf("source directory %q does not exist: %w", c.SourceDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path %q is not a directory", c.SourceDir)
	}
	if c.Canary < 0 || c.Canary > 99 {
		return fmt.Errorf("canary percentage must be between 0 and 99, got %d", c.Canary)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0, got %d", c.Timeout)
	}
	if c.MaxInstances <= 0 {
		return fmt.Errorf("max-instances must be greater than 0, got %d", c.MaxInstances)
	}
	return nil
}

// ParseSecretFlag parses "ENV_VAR=secret-name" into SecretMapping.
func ParseSecretFlag(s string) (SecretMapping, error) {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return SecretMapping{}, fmt.Errorf("invalid secret mapping %q: expected format ENV_VAR=secret-name", s)
	}
	envVar := s[:idx]
	secretName := s[idx+1:]
	if envVar == "" {
		return SecretMapping{}, fmt.Errorf("invalid secret mapping %q: environment variable name is empty", s)
	}
	if secretName == "" {
		return SecretMapping{}, fmt.Errorf("invalid secret mapping %q: secret name is empty", s)
	}
	return SecretMapping{
		EnvVar:     envVar,
		SecretName: secretName,
	}, nil
}
