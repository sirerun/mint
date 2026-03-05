package overlay

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Action represents a single overlay action (update or remove).
type Action struct {
	Target string      `yaml:"target" json:"target"`
	Update interface{} `yaml:"update,omitempty" json:"update,omitempty"`
	Remove bool        `yaml:"remove,omitempty" json:"remove,omitempty"`
}

// Document represents an OpenAPI Overlay document.
type Document struct {
	Overlay string   `yaml:"overlay" json:"overlay"`
	Info    Info     `yaml:"info" json:"info"`
	Actions []Action `yaml:"actions" json:"actions"`
}

// Info holds overlay document metadata.
type Info struct {
	Title   string `yaml:"title" json:"title"`
	Version string `yaml:"version" json:"version"`
}

// Parse parses an overlay document from YAML.
func Parse(data []byte) (*Document, error) {
	var doc Document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing overlay: %w", err)
	}
	if doc.Overlay == "" {
		return nil, fmt.Errorf("missing 'overlay' version field")
	}
	return &doc, nil
}

// Apply applies overlay actions to a target spec (as a generic map).
func Apply(specData []byte, overlay *Document) ([]byte, error) {
	var spec interface{}
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	for i, action := range overlay.Actions {
		if action.Target == "" {
			return nil, fmt.Errorf("action %d: missing target", i)
		}

		if action.Remove {
			spec = removeAtPath(spec, action.Target)
		} else if action.Update != nil {
			spec = updateAtPath(spec, action.Target, action.Update)
		}
	}

	return yaml.Marshal(spec)
}

// updateAtPath sets a value at a JSONPath-like path in a nested structure.
// Supports simple dot-notation paths like "$.info.title" or "$.paths./pets.get.summary".
func updateAtPath(root interface{}, path string, value interface{}) interface{} {
	parts := parsePath(path)
	if len(parts) == 0 {
		return value
	}

	m, ok := root.(map[string]interface{})
	if !ok {
		return root
	}

	if len(parts) == 1 {
		m[parts[0]] = value
		return m
	}

	child, exists := m[parts[0]]
	if !exists {
		child = make(map[string]interface{})
	}

	m[parts[0]] = updateAtPath(child, joinPath(parts[1:]), value)
	return m
}

// removeAtPath removes a value at a JSONPath-like path.
func removeAtPath(root interface{}, path string) interface{} {
	parts := parsePath(path)
	if len(parts) == 0 {
		return root
	}

	m, ok := root.(map[string]interface{})
	if !ok {
		return root
	}

	if len(parts) == 1 {
		delete(m, parts[0])
		return m
	}

	if child, exists := m[parts[0]]; exists {
		m[parts[0]] = removeAtPath(child, joinPath(parts[1:]))
	}

	return m
}

// parsePath splits a JSONPath-like string into segments.
// "$.info.title" -> ["info", "title"]
// "$.paths./pets.get" -> ["paths", "/pets", "get"]
func parsePath(path string) []string {
	// Remove leading "$."
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")

	if path == "" {
		return nil
	}

	// Split carefully: segments starting with / are path keys
	var parts []string
	current := ""
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			if i+1 < len(path) && path[i+1] == '/' {
				// Next segment is a path like /pets
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
				continue
			}
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			continue
		}
		current += string(path[i])
	}
	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func joinPath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	result := "$"
	for _, p := range parts {
		result += "." + p
	}
	return result
}
