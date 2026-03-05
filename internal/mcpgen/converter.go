package mcpgen

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Convert transforms a parsed OpenAPI v3 document into an MCPServer model.
func Convert(doc *v3high.Document) (*MCPServer, error) {
	server := &MCPServer{
		Name:        sanitizeName(doc.Info.Title),
		Version:     doc.Info.Version,
		Description: doc.Info.Description,
	}

	if len(doc.Servers) > 0 {
		server.BaseURL = doc.Servers[0].URL
	}

	server.Auth = extractAuth(doc)

	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return server, nil
	}

	for pair := doc.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		path := pair.Key
		item := pair.Value
		tools := extractToolsFromPathItem(path, item)
		server.Tools = append(server.Tools, tools...)
	}

	return server, nil
}

func extractToolsFromPathItem(path string, item *v3high.PathItem) []MCPTool {
	var tools []MCPTool

	methods := []struct {
		method string
		op     *v3high.Operation
	}{
		{"GET", item.Get},
		{"POST", item.Post},
		{"PUT", item.Put},
		{"DELETE", item.Delete},
		{"PATCH", item.Patch},
		{"HEAD", item.Head},
		{"OPTIONS", item.Options},
	}

	for _, m := range methods {
		if m.op == nil {
			continue
		}
		tool := convertOperation(path, m.method, m.op)
		tools = append(tools, tool)
	}

	return tools
}

func convertOperation(path, method string, op *v3high.Operation) MCPTool {
	name := deriveToolName(op.OperationId, method, path)

	description := op.Summary
	if description == "" {
		description = op.Description
	}
	if description == "" {
		description = "No description"
	}

	tool := MCPTool{
		Name:         name,
		Description:  description,
		HTTPMethod:   method,
		HTTPPath:     path,
		ResponseType: "application/json",
	}

	schema := JSONSchema{
		Type:       "object",
		Properties: make(map[string]JSONSchema),
	}

	for _, param := range op.Parameters {
		p := convertParameter(param)
		tool.Params = append(tool.Params, p)

		propSchema := paramToJSONSchema(param)
		schema.Properties[p.Name] = propSchema
		if p.Required {
			schema.Required = append(schema.Required, p.Name)
		}
	}

	if op.RequestBody != nil {
		bodyParams := extractBodyParams(op.RequestBody)
		for _, p := range bodyParams {
			tool.Params = append(tool.Params, p)
			propSchema := JSONSchema{
				Type:        p.Type,
				Description: p.Description,
				Format:      p.Format,
			}
			if p.Enum != nil {
				for _, e := range p.Enum {
					propSchema.Enum = append(propSchema.Enum, e)
				}
			}
			if p.Items != nil {
				propSchema.Items = p.Items
			}
			schema.Properties[p.Name] = propSchema
			if p.Required {
				schema.Required = append(schema.Required, p.Name)
			}
		}
	}

	tool.InputSchema = schema
	return tool
}

func convertParameter(param *v3high.Parameter) MCPToolParam {
	p := MCPToolParam{
		Name:        param.Name,
		Description: param.Description,
		Required:    derefBool(param.Required),
		In:          param.In,
	}

	if param.Schema != nil {
		s := param.Schema.Schema()
		if s != nil {
			if len(s.Type) > 0 {
				p.Type = s.Type[0]
			}
			p.Format = s.Format
			for _, e := range s.Enum {
				p.Enum = append(p.Enum, fmt.Sprintf("%v", e.Value))
			}
			if s.Default != nil {
				p.Default = s.Default.Value
			}
		}
	}

	if p.Type == "" {
		p.Type = "string"
	}

	return p
}

func paramToJSONSchema(param *v3high.Parameter) JSONSchema {
	js := JSONSchema{
		Description: param.Description,
	}

	if param.Schema != nil {
		s := param.Schema.Schema()
		if s != nil {
			js = schemaToJSONSchema(s)
			if js.Description == "" {
				js.Description = param.Description
			}
		}
	}

	if js.Type == "" {
		js.Type = "string"
	}

	return js
}

func extractBodyParams(reqBody *v3high.RequestBody) []MCPToolParam {
	var params []MCPToolParam

	if reqBody.Content == nil {
		return nil
	}

	// Prefer application/json
	var mediaType *v3high.MediaType
	for pair := reqBody.Content.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Key == "application/json" {
			mediaType = pair.Value
			break
		}
	}

	if mediaType == nil {
		// Take the first content type
		first := reqBody.Content.Oldest()
		if first != nil {
			mediaType = first.Value
		}
	}

	if mediaType == nil || mediaType.Schema == nil {
		return nil
	}

	s := mediaType.Schema.Schema()
	if s == nil {
		return nil
	}

	isRequired := derefBool(reqBody.Required)

	if s.Properties != nil {
		requiredSet := make(map[string]bool)
		for _, r := range s.Required {
			requiredSet[r] = true
		}

		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			propName := pair.Key
			propProxy := pair.Value

			p := MCPToolParam{
				Name:     propName,
				In:       "body",
				Required: isRequired && requiredSet[propName],
			}

			propSchema := propProxy.Schema()
			if propSchema != nil {
				if len(propSchema.Type) > 0 {
					p.Type = propSchema.Type[0]
				}
				p.Description = propSchema.Description
				p.Format = propSchema.Format
				for _, e := range propSchema.Enum {
					p.Enum = append(p.Enum, fmt.Sprintf("%v", e.Value))
				}
				if propSchema.Default != nil {
					p.Default = propSchema.Default.Value
				}
				if p.Type == "array" && propSchema.Items != nil && propSchema.Items.A != nil {
					itemSchema := propSchema.Items.A.Schema()
					if itemSchema != nil {
						js := schemaToJSONSchema(itemSchema)
						p.Items = &js
					}
				}
			}

			if p.Type == "" {
				p.Type = "string"
			}

			params = append(params, p)
		}
	} else {
		// Non-object body: treat the whole body as a single parameter
		p := MCPToolParam{
			Name:     "body",
			In:       "body",
			Required: isRequired,
		}
		if len(s.Type) > 0 {
			p.Type = s.Type[0]
		}
		if p.Type == "" {
			p.Type = "string"
		}
		p.Description = s.Description
		params = append(params, p)
	}

	return params
}

// schemaToJSONSchema converts an OpenAPI schema to a simplified JSONSchema.
func schemaToJSONSchema(s *base.Schema) JSONSchema {
	js := JSONSchema{
		Description: s.Description,
		Format:      s.Format,
	}

	if len(s.Type) > 0 {
		js.Type = s.Type[0]
	}

	for _, e := range s.Enum {
		js.Enum = append(js.Enum, e.Value)
	}

	if s.Default != nil {
		js.Default = s.Default.Value
	}

	if js.Type == "object" && s.Properties != nil {
		js.Properties = make(map[string]JSONSchema)
		for pair := s.Properties.Oldest(); pair != nil; pair = pair.Next() {
			propSchema := pair.Value.Schema()
			if propSchema != nil {
				js.Properties[pair.Key] = schemaToJSONSchema(propSchema)
			}
		}
		js.Required = s.Required
	}

	if js.Type == "array" && s.Items != nil && s.Items.A != nil {
		itemSchema := s.Items.A.Schema()
		if itemSchema != nil {
			itemJS := schemaToJSONSchema(itemSchema)
			js.Items = &itemJS
		}
	}

	if js.Type == "" {
		js.Type = "string"
	}

	return js
}

func deriveToolName(operationID, method, path string) string {
	if operationID != "" {
		return toSnakeCase(operationID)
	}

	// Derive from method + path: GET /users/{id} -> get_users_by_id
	clean := strings.NewReplacer(
		"{", "",
		"}", "",
		"/", "_",
	).Replace(path)

	clean = strings.Trim(clean, "_")
	name := strings.ToLower(method) + "_" + clean

	// Collapse multiple underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	return name
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					result.WriteByte('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func sanitizeName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			result.WriteRune(unicode.ToLower(r))
		} else if r == ' ' {
			result.WriteByte('-')
		}
	}
	return result.String()
}

func extractAuth(doc *v3high.Document) *MCPAuth {
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return nil
	}

	for pair := doc.Components.SecuritySchemes.Oldest(); pair != nil; pair = pair.Next() {
		scheme := pair.Value
		switch scheme.Type {
		case "apiKey":
			return &MCPAuth{
				Type:       "apiKey",
				HeaderName: scheme.Name,
				EnvVar:     "MINT_API_KEY",
			}
		case "http":
			if strings.EqualFold(scheme.Scheme, "bearer") {
				return &MCPAuth{
					Type:       "bearer",
					HeaderName: "Authorization",
					EnvVar:     "MINT_TOKEN",
				}
			}
		case "oauth2":
			return &MCPAuth{
				Type:       "oauth2",
				HeaderName: "Authorization",
				EnvVar:     "MINT_TOKEN",
			}
		}
	}

	return nil
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
