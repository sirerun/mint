package registry

// Verification records the functional-verification state of a registry entry.
// It is populated by the mcp-registry verification harness (see ADR-179/180).
// Tier is one of "t0".."t5"; the remaining fields are optional and only set
// once an entry has been through the harness.
type Verification struct {
	Tier            string `json:"tier"`
	VerifiedAt      string `json:"verified_at,omitempty"`
	VerifierVersion string `json:"verifier_version,omitempty"`
	FailureCode     string `json:"failure_code,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

// RegistryEntry represents a single MCP server in the registry.
type RegistryEntry struct {
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Tags           []string      `json:"tags"`
	SpecURL        string        `json:"spec_url"`
	AuthType       string        `json:"auth_type"`
	AuthEnvVar     string        `json:"auth_env_var"`
	MinMintVersion string        `json:"min_mint_version"`
	Verification   *Verification `json:"verification,omitempty"`
}

// RegistryIndex represents the full registry index.
//
// The on-disk registry.json uses the keys {schema_version, apis}; the in-memory
// field names stay Version/Entries. The JSON tags map the two so mint parses the
// real on-disk shape directly (previously the tags read {version, entries}, which
// silently produced an empty index against the real catalog).
type RegistryIndex struct {
	Version int             `json:"schema_version"`
	Entries []RegistryEntry `json:"apis"`
}
