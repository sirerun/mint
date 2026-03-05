package mcpgen

// MCPServer represents a generated MCP server with its tools and configuration.
type MCPServer struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	BaseURL     string    `json:"base_url"`
	Tools       []MCPTool `json:"tools"`
	Auth        *MCPAuth  `json:"auth,omitempty"`
}

// MCPTool represents a single MCP tool derived from an OpenAPI operation.
type MCPTool struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	HTTPMethod   string         `json:"http_method"`
	HTTPPath     string         `json:"http_path"`
	InputSchema  JSONSchema     `json:"input_schema"`
	ResponseType string         `json:"response_type"`
	Params       []MCPToolParam `json:"params"`
}

// MCPToolParam describes a single parameter for an MCP tool.
type MCPToolParam struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	In          string      `json:"in"` // path, query, header, body
	Enum        []string    `json:"enum,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Format      string      `json:"format,omitempty"`
	Items       *JSONSchema `json:"items,omitempty"`
}

// MCPAuth describes authentication configuration for the generated server.
type MCPAuth struct {
	Type       string `json:"type"`        // apiKey, bearer, oauth2
	HeaderName string `json:"header_name"` // e.g. X-API-Key, Authorization
	EnvVar     string `json:"env_var"`     // e.g. MINT_API_KEY
}

// JSONSchema is a simplified JSON Schema representation for MCP tool input schemas.
type JSONSchema struct {
	Type        string                `json:"type"`
	Description string                `json:"description,omitempty"`
	Properties  map[string]JSONSchema `json:"properties,omitempty"`
	Required    []string              `json:"required,omitempty"`
	Items       *JSONSchema           `json:"items,omitempty"`
	Enum        []interface{}         `json:"enum,omitempty"`
	Default     interface{}           `json:"default,omitempty"`
	Format      string                `json:"format,omitempty"`
}
