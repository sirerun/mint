package lint

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Severity levels for lint diagnostics.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Diagnostic represents a single lint finding.
type Diagnostic struct {
	Severity string `json:"severity"`
	RuleID   string `json:"rule_id"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
}

func (d Diagnostic) String() string {
	if d.Path != "" {
		return fmt.Sprintf("[%s] %s: %s (%s)", d.Severity, d.Path, d.Message, d.RuleID)
	}
	return fmt.Sprintf("[%s] %s (%s)", d.Severity, d.Message, d.RuleID)
}

// Result holds all diagnostics from a lint run.
type Result struct {
	Errors   int          `json:"errors"`
	Warnings int          `json:"warnings"`
	Infos    int          `json:"infos"`
	Items    []Diagnostic `json:"items"`
}

func (r *Result) add(d Diagnostic) {
	r.Items = append(r.Items, d)
	switch d.Severity {
	case SeverityError:
		r.Errors++
	case SeverityWarning:
		r.Warnings++
	case SeverityInfo:
		r.Infos++
	}
}

// Ruleset defines which rules are enabled and their severity.
type Ruleset struct {
	Name  string
	Rules map[string]string // rule ID → severity (error, warning, info, off)
}

// PredefinedRulesets returns the built-in rulesets.
func PredefinedRulesets() map[string]Ruleset {
	return map[string]Ruleset{
		"minimal": {
			Name: "minimal",
			Rules: map[string]string{
				"info-required":     SeverityError,
				"info-title":        SeverityWarning,
				"info-version":      SeverityWarning,
				"paths-defined":     SeverityWarning,
				"operation-id":      "off",
				"operation-desc":    "off",
				"operation-resp":    "off",
				"param-description": "off",
				"tag-description":   "off",
				"no-empty-servers":  "off",
				"contact-defined":   "off",
				"license-defined":   "off",
			},
		},
		"recommended": {
			Name: "recommended",
			Rules: map[string]string{
				"info-required":     SeverityError,
				"info-title":        SeverityWarning,
				"info-version":      SeverityWarning,
				"paths-defined":     SeverityWarning,
				"operation-id":      SeverityWarning,
				"operation-desc":    SeverityInfo,
				"operation-resp":    SeverityWarning,
				"param-description": SeverityInfo,
				"tag-description":   "off",
				"no-empty-servers":  SeverityInfo,
				"contact-defined":   "off",
				"license-defined":   "off",
			},
		},
		"strict": {
			Name: "strict",
			Rules: map[string]string{
				"info-required":     SeverityError,
				"info-title":        SeverityError,
				"info-version":      SeverityError,
				"paths-defined":     SeverityError,
				"operation-id":      SeverityError,
				"operation-desc":    SeverityWarning,
				"operation-resp":    SeverityError,
				"param-description": SeverityWarning,
				"tag-description":   SeverityWarning,
				"no-empty-servers":  SeverityWarning,
				"contact-defined":   SeverityInfo,
				"license-defined":   SeverityInfo,
			},
		},
	}
}

// GetRuleset returns a predefined ruleset by name.
func GetRuleset(name string) (Ruleset, bool) {
	rs, ok := PredefinedRulesets()[name]
	return rs, ok
}

// Run lints the given OpenAPI spec data using the provided ruleset.
func Run(data []byte, source string, rs Ruleset) (*Result, error) {
	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
	}

	doc, err := libopenapi.NewDocumentWithConfiguration(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}

	model, buildErr := doc.BuildV3Model()
	if buildErr != nil {
		return nil, fmt.Errorf("failed to build model: %w", buildErr)
	}

	if model == nil {
		return nil, fmt.Errorf("spec produced no model")
	}

	result := &Result{}
	checkInfoRules(&model.Model, result, source, rs)
	checkPathRules(&model.Model, result, source, rs)
	checkOperationRules(&model.Model, result, source, rs)
	checkServerRules(&model.Model, result, source, rs)

	return result, nil
}

func ruleSeverity(rs Ruleset, ruleID string) string {
	if sev, ok := rs.Rules[ruleID]; ok {
		return sev
	}
	return "off"
}

func checkInfoRules(doc *v3high.Document, result *Result, source string, rs Ruleset) {
	if doc.Info == nil {
		if sev := ruleSeverity(rs, "info-required"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "info-required",
				Message:  "missing required field: info",
				Path:     source,
			})
		}
		return
	}
	if doc.Info.Title == "" {
		if sev := ruleSeverity(rs, "info-title"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "info-title",
				Message:  "info.title is empty",
				Path:     source,
			})
		}
	}
	if doc.Info.Version == "" {
		if sev := ruleSeverity(rs, "info-version"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "info-version",
				Message:  "info.version is empty",
				Path:     source,
			})
		}
	}
	if doc.Info.Contact == nil {
		if sev := ruleSeverity(rs, "contact-defined"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "contact-defined",
				Message:  "info.contact is not defined",
				Path:     source,
			})
		}
	}
	if doc.Info.License == nil {
		if sev := ruleSeverity(rs, "license-defined"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "license-defined",
				Message:  "info.license is not defined",
				Path:     source,
			})
		}
	}
}

func checkPathRules(doc *v3high.Document, result *Result, source string, rs Ruleset) {
	if doc.Paths == nil || doc.Paths.PathItems == nil || doc.Paths.PathItems.Len() == 0 {
		if sev := ruleSeverity(rs, "paths-defined"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "paths-defined",
				Message:  "spec has no paths defined",
				Path:     source,
			})
		}
	}
}

func checkOperationRules(doc *v3high.Document, result *Result, source string, rs Ruleset) {
	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return
	}

	for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		path := pair.Key
		item := pair.Value

		ops := []*v3high.Operation{
			item.Get, item.Post, item.Put, item.Delete, item.Patch,
		}
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for i, op := range ops {
			if op == nil {
				continue
			}
			if op.OperationId == "" {
				if sev := ruleSeverity(rs, "operation-id"); sev != "off" {
					result.add(Diagnostic{
						Severity: sev,
						RuleID:   "operation-id",
						Message:  fmt.Sprintf("%s %s: missing operationId", methods[i], path),
						Path:     source,
					})
				}
			}
			if op.Summary == "" && op.Description == "" {
				if sev := ruleSeverity(rs, "operation-desc"); sev != "off" {
					result.add(Diagnostic{
						Severity: sev,
						RuleID:   "operation-desc",
						Message:  fmt.Sprintf("%s %s: missing summary and description", methods[i], path),
						Path:     source,
					})
				}
			}
			if op.Responses == nil || op.Responses.Codes == nil || op.Responses.Codes.Len() == 0 {
				if sev := ruleSeverity(rs, "operation-resp"); sev != "off" {
					result.add(Diagnostic{
						Severity: sev,
						RuleID:   "operation-resp",
						Message:  fmt.Sprintf("%s %s: no response codes defined", methods[i], path),
						Path:     source,
					})
				}
			}
			checkParamRules(op, path, methods[i], result, source, rs)
		}
	}
}

func checkParamRules(op *v3high.Operation, path, method string, result *Result, source string, rs Ruleset) {
	for _, param := range op.Parameters {
		if param == nil {
			continue
		}
		if param.Description == "" {
			if sev := ruleSeverity(rs, "param-description"); sev != "off" {
				result.add(Diagnostic{
					Severity: sev,
					RuleID:   "param-description",
					Message:  fmt.Sprintf("%s %s: parameter %q has no description", method, path, param.Name),
					Path:     source,
				})
			}
		}
	}
}

func checkServerRules(doc *v3high.Document, result *Result, source string, rs Ruleset) {
	if len(doc.Servers) == 0 {
		if sev := ruleSeverity(rs, "no-empty-servers"); sev != "off" {
			result.add(Diagnostic{
				Severity: sev,
				RuleID:   "no-empty-servers",
				Message:  "no servers defined",
				Path:     source,
			})
		}
	}
}
