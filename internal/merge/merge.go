package merge

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"go.yaml.in/yaml/v4"
)

// ConflictStrategy determines how conflicts are handled.
type ConflictStrategy string

const (
	StrategyFail   ConflictStrategy = "fail"
	StrategyRename ConflictStrategy = "rename"
	StrategySkip   ConflictStrategy = "skip"
)

// Conflict describes a merge conflict.
type Conflict struct {
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

func (c Conflict) String() string {
	return fmt.Sprintf("conflict at %s: %s", c.Path, c.Detail)
}

// Result holds the merge output.
type Result struct {
	Output    []byte     `json:"-"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
}

// Specs merges multiple OpenAPI documents into one.
func Specs(specs [][]byte, strategy ConflictStrategy) (*Result, error) {
	if len(specs) < 2 {
		return nil, fmt.Errorf("at least 2 specs required for merging")
	}

	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
	}

	// Parse all specs
	var models []*v3high.Document
	for i, data := range specs {
		doc, err := libopenapi.NewDocumentWithConfiguration(data, &config)
		if err != nil {
			return nil, fmt.Errorf("parsing spec %d: %w", i+1, err)
		}
		m, err := doc.BuildV3Model()
		if err != nil {
			return nil, fmt.Errorf("building model for spec %d: %w", i+1, err)
		}
		models = append(models, &m.Model)
	}

	// Use the first spec as the base
	base := models[0]
	result := &Result{}

	// Merge subsequent specs into base
	for i := 1; i < len(models); i++ {
		conflicts := mergePaths(base, models[i], strategy)
		result.Conflicts = append(result.Conflicts, conflicts...)
	}

	if strategy == StrategyFail && len(result.Conflicts) > 0 {
		return result, fmt.Errorf("merge failed: %d conflict(s) found", len(result.Conflicts))
	}

	// Serialize the merged model
	output, err := serializeMerged(base)
	if err != nil {
		return nil, fmt.Errorf("serializing merged spec: %w", err)
	}
	result.Output = output

	return result, nil
}

func mergePaths(base, other *v3high.Document, strategy ConflictStrategy) []Conflict {
	var conflicts []Conflict

	if other.Paths == nil || other.Paths.PathItems == nil {
		return nil
	}

	if base.Paths == nil {
		base.Paths = other.Paths
		return nil
	}

	for pair := other.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		path := pair.Key
		item := pair.Value

		existing := base.Paths.PathItems.GetOrZero(path)
		if existing != nil {
			conflicts = append(conflicts, Conflict{
				Path:   path,
				Detail: "path exists in both specs",
			})
			if strategy == StrategySkip {
				continue
			}
		}
		base.Paths.PathItems.Set(path, item)
	}

	return conflicts
}

func serializeMerged(doc *v3high.Document) ([]byte, error) {
	// Build a simplified YAML representation
	m := map[string]interface{}{
		"openapi": "3.0.3",
	}

	if doc.Info != nil {
		info := map[string]interface{}{
			"title":   doc.Info.Title,
			"version": doc.Info.Version,
		}
		if doc.Info.Description != "" {
			info["description"] = doc.Info.Description
		}
		m["info"] = info
	}

	if doc.Paths != nil && doc.Paths.PathItems != nil {
		paths := make(map[string]interface{})
		for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
			pathOps := make(map[string]interface{})
			item := pair.Value

			ops := []struct {
				method string
				op     *v3high.Operation
			}{
				{"get", item.Get},
				{"post", item.Post},
				{"put", item.Put},
				{"delete", item.Delete},
				{"patch", item.Patch},
			}

			for _, o := range ops {
				if o.op == nil {
					continue
				}
				opData := map[string]interface{}{}
				if o.op.OperationId != "" {
					opData["operationId"] = o.op.OperationId
				}
				if o.op.Summary != "" {
					opData["summary"] = o.op.Summary
				}
				if o.op.Description != "" {
					opData["description"] = o.op.Description
				}
				opData["responses"] = map[string]interface{}{
					"200": map[string]interface{}{
						"description": "OK",
					},
				}
				pathOps[o.method] = opData
			}
			paths[pair.Key] = pathOps
		}
		m["paths"] = paths
	}

	return yaml.Marshal(m)
}
