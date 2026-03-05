package diff

import (
	"fmt"

	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// ChangeType classifies a change.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Modified ChangeType = "modified"
)

// Change represents a single difference between two specs.
type Change struct {
	Type     ChangeType `json:"type"`
	Path     string     `json:"path"`
	Method   string     `json:"method,omitempty"`
	Detail   string     `json:"detail"`
	Breaking bool       `json:"breaking"`
}

func (c Change) String() string {
	b := ""
	if c.Breaking {
		b = " [BREAKING]"
	}
	if c.Method != "" {
		return fmt.Sprintf("[%s] %s %s: %s%s", c.Type, c.Method, c.Path, c.Detail, b)
	}
	return fmt.Sprintf("[%s] %s: %s%s", c.Type, c.Path, c.Detail, b)
}

// Result holds the diff output.
type Result struct {
	Changes         []Change `json:"changes"`
	HasBreaking     bool     `json:"has_breaking"`
	TotalChanges    int      `json:"total_changes"`
	BreakingChanges int      `json:"breaking_changes"`
}

// Specs compares two OpenAPI v3 documents and returns the differences.
func Specs(old, new *v3high.Document) *Result {
	result := &Result{}

	diffPaths(old, new, result)
	diffInfo(old, new, result)

	result.TotalChanges = len(result.Changes)
	for _, c := range result.Changes {
		if c.Breaking {
			result.HasBreaking = true
			result.BreakingChanges++
		}
	}

	return result
}

func diffInfo(old, new *v3high.Document, result *Result) {
	if old.Info != nil && new.Info != nil {
		if old.Info.Title != new.Info.Title {
			result.Changes = append(result.Changes, Change{
				Type:   Modified,
				Path:   "info.title",
				Detail: fmt.Sprintf("changed from %q to %q", old.Info.Title, new.Info.Title),
			})
		}
		if old.Info.Version != new.Info.Version {
			result.Changes = append(result.Changes, Change{
				Type:   Modified,
				Path:   "info.version",
				Detail: fmt.Sprintf("changed from %q to %q", old.Info.Version, new.Info.Version),
			})
		}
	}
}

func diffPaths(old, new *v3high.Document, result *Result) {
	oldPaths := collectPaths(old)
	newPaths := collectPaths(new)

	// Check for removed paths (breaking)
	for path := range oldPaths {
		if _, ok := newPaths[path]; !ok {
			result.Changes = append(result.Changes, Change{
				Type:     Removed,
				Path:     path,
				Detail:   "path removed",
				Breaking: true,
			})
		}
	}

	// Check for added paths
	for path := range newPaths {
		if _, ok := oldPaths[path]; !ok {
			result.Changes = append(result.Changes, Change{
				Type:   Added,
				Path:   path,
				Detail: "path added",
			})
		}
	}

	// Check operations on shared paths
	if old.Paths != nil && old.Paths.PathItems != nil && new.Paths != nil && new.Paths.PathItems != nil {
		for pair := old.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
			path := pair.Key
			oldItem := pair.Value

			if newPaths[path] == nil {
				continue
			}

			newItem := newPaths[path]
			diffOperations(path, oldItem, newItem, result)
		}
	}
}

func diffOperations(path string, old, new *v3high.PathItem, result *Result) {
	ops := []struct {
		method string
		oldOp  *v3high.Operation
		newOp  *v3high.Operation
	}{
		{"GET", old.Get, new.Get},
		{"POST", old.Post, new.Post},
		{"PUT", old.Put, new.Put},
		{"DELETE", old.Delete, new.Delete},
		{"PATCH", old.Patch, new.Patch},
	}

	for _, o := range ops {
		if o.oldOp != nil && o.newOp == nil {
			result.Changes = append(result.Changes, Change{
				Type:     Removed,
				Path:     path,
				Method:   o.method,
				Detail:   "operation removed",
				Breaking: true,
			})
		}
		if o.oldOp == nil && o.newOp != nil {
			result.Changes = append(result.Changes, Change{
				Type:   Added,
				Path:   path,
				Method: o.method,
				Detail: "operation added",
			})
		}
		if o.oldOp != nil && o.newOp != nil {
			diffOperation(path, o.method, o.oldOp, o.newOp, result)
		}
	}
}

func diffOperation(path, method string, old, new *v3high.Operation, result *Result) {
	// Check for added required parameters (breaking)
	oldParams := make(map[string]*v3high.Parameter)
	for _, p := range old.Parameters {
		oldParams[p.Name] = p
	}

	for _, p := range new.Parameters {
		if _, ok := oldParams[p.Name]; !ok {
			breaking := derefBool(p.Required)
			result.Changes = append(result.Changes, Change{
				Type:     Added,
				Path:     path,
				Method:   method,
				Detail:   fmt.Sprintf("parameter %q added", p.Name),
				Breaking: breaking,
			})
		}
	}

	// Check for removed parameters (breaking)
	newParams := make(map[string]*v3high.Parameter)
	for _, p := range new.Parameters {
		newParams[p.Name] = p
	}

	for _, p := range old.Parameters {
		if _, ok := newParams[p.Name]; !ok {
			result.Changes = append(result.Changes, Change{
				Type:     Removed,
				Path:     path,
				Method:   method,
				Detail:   fmt.Sprintf("parameter %q removed", p.Name),
				Breaking: true,
			})
		}
	}
}

func collectPaths(doc *v3high.Document) map[string]*v3high.PathItem {
	paths := make(map[string]*v3high.PathItem)
	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return paths
	}
	for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		paths[pair.Key] = pair.Value
	}
	return paths
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
