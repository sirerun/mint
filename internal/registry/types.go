package registry

// RegistryEntry represents a single MCP server in the registry.
type RegistryEntry struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	SpecURL        string   `json:"spec_url"`
	AuthType       string   `json:"auth_type"`
	AuthEnvVar     string   `json:"auth_env_var"`
	MinMintVersion string   `json:"min_mint_version"`
}

// RegistryIndex represents the full registry index.
type RegistryIndex struct {
	Version int             `json:"version"`
	Entries []RegistryEntry `json:"entries"`
}
