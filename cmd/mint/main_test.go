package main

import (
	"os"
	"testing"
)

func TestRunHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "no args", args: nil, want: 0},
		{name: "help", args: []string{"help"}, want: 0},
		{name: "-h", args: []string{"-h"}, want: 0},
		{name: "--help", args: []string{"--help"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.args); got != tt.want {
				t.Errorf("run(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "version", args: []string{"version"}},
		{name: "-v", args: []string{"-v"}},
		{name: "--version", args: []string{"--version"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.args); got != 0 {
				t.Errorf("run(%v) = %d, want 0", tt.args, got)
			}
		})
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if got := run([]string{"notacommand"}); got != 1 {
		t.Errorf("run([notacommand]) = %d, want 1", got)
	}
}

func TestRunNewCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "login no github", args: []string{"login"}, want: 1},
		{name: "login help", args: []string{"login", "--help"}, want: 0},
		{name: "publish help", args: []string{"publish", "--help"}, want: 0},
		{name: "publish no manifest", args: []string{"publish", "--dry-run", "--dir", t.TempDir()}, want: 1},
		{name: "install no args", args: []string{"install"}, want: 1},
		{name: "install help", args: []string{"install", "--help"}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.args); got != tt.want {
				t.Errorf("run(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestRunPublishDryRun(t *testing.T) {
	dir := t.TempDir()
	manifest := `{"name":"test-server","version":"1.0.0","description":"A test"}`
	os.WriteFile(dir+"/mint.json", []byte(manifest), 0o644)

	if got := run([]string{"publish", "--dry-run", "--dir", dir}); got != 0 {
		t.Errorf("publish --dry-run = %d, want 0", got)
	}
}

func TestRunSubcommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{name: "mcp help", args: []string{"mcp"}, want: 0},
		{name: "mcp unknown", args: []string{"mcp", "notacommand"}, want: 1},
		{name: "lint no args", args: []string{"lint"}, want: 1},
		{name: "validate no args", args: []string{"validate"}, want: 1},
		{name: "diff no args", args: []string{"diff"}, want: 1},
		{name: "merge no args", args: []string{"merge"}, want: 1},
		{name: "mcp generate no args", args: []string{"mcp", "generate"}, want: 1},
		{name: "deploy no args", args: []string{"deploy"}, want: 0},
		{name: "deploy help", args: []string{"deploy", "--help"}, want: 0},
		{name: "deploy -h", args: []string{"deploy", "-h"}, want: 0},
		{name: "deploy gcp --help", args: []string{"deploy", "gcp", "--help"}, want: 0},
		{name: "deploy status --help", args: []string{"deploy", "status", "--help"}, want: 0},
		{name: "deploy rollback --help", args: []string{"deploy", "rollback", "--help"}, want: 0},
		{name: "deploy unknown", args: []string{"deploy", "notacommand"}, want: 1},
		{name: "deploy gcp no source", args: []string{"deploy", "gcp", "--project", "test"}, want: 1},
		{name: "deploy status stub", args: []string{"deploy", "status"}, want: 1},
		{name: "deploy rollback stub", args: []string{"deploy", "rollback"}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.args); got != tt.want {
				t.Errorf("run(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}
