package validate

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	validator "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Diagnostic represents a single validation finding.
type Diagnostic struct {
	Severity string `json:"severity"` // error, warning, info
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	RuleID   string `json:"rule_id,omitempty"`
}

func (d Diagnostic) String() string {
	prefix := d.Severity
	if d.Path != "" {
		return fmt.Sprintf("[%s] %s: %s", prefix, d.Path, d.Message)
	}
	return fmt.Sprintf("[%s] %s", prefix, d.Message)
}

// Result holds all diagnostics from a validation run.
type Result struct {
	Valid       bool         `json:"valid"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Spec validates an OpenAPI specification for structural correctness.
func Spec(data []byte, source string) *Result {
	result := &Result{Valid: true}

	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
	}

	doc, err := libopenapi.NewDocumentWithConfiguration(data, &config)
	if err != nil {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Message:  fmt.Sprintf("failed to parse: %v", err),
			Path:     source,
		})
		return result
	}

	model, buildErr := doc.BuildV3Model()
	if buildErr != nil {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Message:  buildErr.Error(),
			Path:     source,
		})
		return result
	}

	if model == nil {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Message:  "spec produced no model",
			Path:     source,
		})
		return result
	}

	checkInfo(&model.Model, result, source)
	checkPaths(&model.Model, result, source)
	checkOperations(&model.Model, result, source)

	return result
}

func checkInfo(doc *validator.Document, result *Result, source string) {
	if doc.Info == nil {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "error",
			Message:  "missing required field: info",
			Path:     source,
			RuleID:   "info-required",
		})
		return
	}
	if doc.Info.Title == "" {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "warning",
			Message:  "info.title is empty",
			Path:     source,
			RuleID:   "info-title",
		})
	}
	if doc.Info.Version == "" {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "warning",
			Message:  "info.version is empty",
			Path:     source,
			RuleID:   "info-version",
		})
	}
}

func checkPaths(doc *validator.Document, result *Result, source string) {
	if doc.Paths == nil || doc.Paths.PathItems == nil || doc.Paths.PathItems.Len() == 0 {
		result.Diagnostics = append(result.Diagnostics, Diagnostic{
			Severity: "warning",
			Message:  "spec has no paths defined",
			Path:     source,
			RuleID:   "paths-defined",
		})
	}
}

func checkOperations(doc *validator.Document, result *Result, source string) {
	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return
	}

	for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		path := pair.Key
		item := pair.Value

		ops := []*validator.Operation{
			item.Get, item.Post, item.Put, item.Delete, item.Patch,
		}
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for i, op := range ops {
			if op == nil {
				continue
			}
			if op.OperationId == "" {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: "warning",
					Message:  fmt.Sprintf("%s %s: missing operationId", methods[i], path),
					Path:     source,
					RuleID:   "operation-id",
				})
			}
			if op.Summary == "" && op.Description == "" {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: "info",
					Message:  fmt.Sprintf("%s %s: missing summary and description", methods[i], path),
					Path:     source,
					RuleID:   "operation-description",
				})
			}
			if op.Responses == nil || op.Responses.Codes == nil || op.Responses.Codes.Len() == 0 {
				result.Diagnostics = append(result.Diagnostics, Diagnostic{
					Severity: "warning",
					Message:  fmt.Sprintf("%s %s: no response codes defined", methods[i], path),
					Path:     source,
					RuleID:   "operation-responses",
				})
			}
		}
	}
}
