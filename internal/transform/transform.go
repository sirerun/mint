package transform

import (
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

// FilterOperations filters a spec to only include operations matching the given tags or path patterns.
func FilterOperations(specData []byte, includeTags []string, excludePaths []string) ([]byte, error) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		return yaml.Marshal(spec)
	}

	tagSet := make(map[string]bool)
	for _, t := range includeTags {
		tagSet[strings.TrimSpace(t)] = true
	}

	filtered := make(map[string]interface{})
	for path, item := range paths {
		if matchesExclude(path, excludePaths) {
			continue
		}

		if len(tagSet) == 0 {
			filtered[path] = item
			continue
		}

		ops, ok := item.(map[string]interface{})
		if !ok {
			filtered[path] = item
			continue
		}

		filteredOps := make(map[string]interface{})
		for method, op := range ops {
			opMap, ok := op.(map[string]interface{})
			if !ok {
				filteredOps[method] = op
				continue
			}
			if hasMatchingTag(opMap, tagSet) {
				filteredOps[method] = op
			}
		}
		if len(filteredOps) > 0 {
			filtered[path] = filteredOps
		}
	}

	spec["paths"] = filtered
	return yaml.Marshal(spec)
}

// Normalize sorts keys and applies consistent formatting to a spec.
func Normalize(specData []byte) ([]byte, error) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	sorted := sortKeys(spec)
	return yaml.Marshal(sorted)
}

// RemoveUnusedComponents removes components not referenced by any operation.
func RemoveUnusedComponents(specData []byte) ([]byte, error) {
	var spec map[string]interface{}
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}

	raw := string(specData)
	components, ok := spec["components"].(map[string]interface{})
	if !ok {
		return yaml.Marshal(spec)
	}

	schemas, ok := components["schemas"].(map[string]interface{})
	if ok {
		cleaned := make(map[string]interface{})
		for name, schema := range schemas {
			ref := fmt.Sprintf("#/components/schemas/%s", name)
			if strings.Contains(raw, ref) {
				cleaned[name] = schema
			}
		}
		if len(cleaned) > 0 {
			components["schemas"] = cleaned
		} else {
			delete(components, "schemas")
		}
	}

	if len(components) > 0 {
		spec["components"] = components
	} else {
		delete(spec, "components")
	}

	return yaml.Marshal(spec)
}

func matchesExclude(path string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "*") {
			if strings.HasPrefix(path, strings.TrimSuffix(p, "*")) {
				return true
			}
		} else if path == p {
			return true
		}
	}
	return false
}

func hasMatchingTag(opMap map[string]interface{}, tagSet map[string]bool) bool {
	tags, ok := opMap["tags"]
	if !ok {
		return false
	}

	tagList, ok := tags.([]interface{})
	if !ok {
		return false
	}

	for _, tag := range tagList {
		if s, ok := tag.(string); ok && tagSet[s] {
			return true
		}
	}
	return false
}

func sortKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		if sub, ok := v.(map[string]interface{}); ok {
			result[k] = sortKeys(sub)
		} else {
			result[k] = v
		}
	}
	return result
}
