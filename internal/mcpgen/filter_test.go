package mcpgen

import (
	"testing"
)

func TestFilterByPaths(t *testing.T) {
	tools := []MCPTool{
		{Name: "list_users", HTTPPath: "/users"},
		{Name: "get_user", HTTPPath: "/users/{id}"},
		{Name: "list_internal", HTTPPath: "/internal/metrics"},
		{Name: "get_health", HTTPPath: "/internal/health"},
		{Name: "list_pets", HTTPPath: "/pets"},
	}

	tests := []struct {
		name     string
		patterns []string
		want     int
	}{
		{"exclude prefix", []string{"/internal/*"}, 3},
		{"exclude exact", []string{"/pets"}, 4},
		{"exclude multiple", []string{"/internal/*", "/pets"}, 2},
		{"no match", []string{"/nonexistent"}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByPaths(tools, tt.patterns)
			if len(got) != tt.want {
				t.Errorf("FilterByPaths() returned %d tools, want %d", len(got), tt.want)
			}
		})
	}
}

func TestMatchesAnyPattern(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"/users", []string{"/users"}, true},
		{"/users/1", []string{"/users"}, false},
		{"/internal/foo", []string{"/internal/*"}, true},
		{"/internal", []string{"/internal/*"}, false},
		{"/users", []string{"/pets"}, false},
	}

	for _, tt := range tests {
		if got := matchesAnyPattern(tt.path, tt.patterns); got != tt.want {
			t.Errorf("matchesAnyPattern(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
		}
	}
}
