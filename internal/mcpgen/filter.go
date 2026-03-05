package mcpgen

import (
	"strings"

	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// FilterByTags returns only tools whose corresponding operations have at least one of the given tags.
func FilterByTags(tools []MCPTool, tags []string, doc *v3high.Document) []MCPTool {
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[strings.TrimSpace(t)] = true
	}

	// Build a map of operationId -> tags from the spec
	opTags := buildOperationTagMap(doc)

	var filtered []MCPTool
	for _, tool := range tools {
		toolTags := opTags[tool.Name]
		for _, tt := range toolTags {
			if tagSet[tt] {
				filtered = append(filtered, tool)
				break
			}
		}
	}

	return filtered
}

// FilterByPaths removes tools whose HTTP path matches any of the given patterns.
// Patterns support simple prefix matching with trailing *.
func FilterByPaths(tools []MCPTool, patterns []string) []MCPTool {
	var filtered []MCPTool
	for _, tool := range tools {
		if !matchesAnyPattern(tool.HTTPPath, patterns) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

func matchesAnyPattern(path string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if path == p {
			return true
		}
	}
	return false
}

func buildOperationTagMap(doc *v3high.Document) map[string][]string {
	result := make(map[string][]string)

	if doc == nil || doc.Paths == nil || doc.Paths.PathItems == nil {
		return result
	}

	for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		path := pair.Key
		item := pair.Value

		ops := []struct {
			method string
			op     *v3high.Operation
		}{
			{"GET", item.Get},
			{"POST", item.Post},
			{"PUT", item.Put},
			{"DELETE", item.Delete},
			{"PATCH", item.Patch},
		}

		for _, o := range ops {
			if o.op == nil {
				continue
			}
			name := deriveToolName(o.op.OperationId, o.method, path)
			result[name] = o.op.Tags
		}
	}

	return result
}
