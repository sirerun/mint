package gcp

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecGitClient implements GitClient by shelling out to the git binary.
type ExecGitClient struct{}

// Compile-time interface check.
var _ GitClient = (*ExecGitClient)(nil)

// NewExecGitClient returns a new ExecGitClient after verifying that git is in PATH.
func NewExecGitClient() (*ExecGitClient, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git not found in PATH: %w", err)
	}
	return &ExecGitClient{}, nil
}

func (g *ExecGitClient) Init(dir string) error {
	return g.run(dir, "init")
}

func (g *ExecGitClient) AddAll(dir string) error {
	return g.run(dir, "add", "-A")
}

func (g *ExecGitClient) Commit(dir string, message string) error {
	return g.run(dir, "commit", "-m", message, "--allow-empty")
}

func (g *ExecGitClient) AddRemote(dir string, name, url string) error {
	return g.run(dir, "remote", "add", name, url)
}

func (g *ExecGitClient) Push(dir string, remote, branch string) error {
	return g.run(dir, "push", remote, branch)
}

func (g *ExecGitClient) HasRemote(dir string, name string) (bool, error) {
	out, err := g.output(dir, "remote")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

func (g *ExecGitClient) run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (g *ExecGitClient) output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}
