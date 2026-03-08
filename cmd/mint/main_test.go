package main

import (
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
		{name: "deploy gcp stub", args: []string{"deploy", "gcp"}, want: 1},
		{name: "deploy status stub", args: []string{"deploy", "status"}, want: 1},
		{name: "deploy rollback stub", args: []string{"deploy", "rollback"}, want: 1},
		{name: "deploy unknown", args: []string{"deploy", "unknowncmd"}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.args); got != tt.want {
				t.Errorf("run(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}
