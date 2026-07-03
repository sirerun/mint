package registry

import (
	"encoding/json"
	"os"
	"testing"
)

func TestRegistryEntry_JSONRoundTrip(t *testing.T) {
	entry := RegistryEntry{
		Name:           "github",
		Description:    "GitHub API v3",
		Tags:           []string{"scm", "api"},
		SpecURL:        "https://example.com/github.yaml",
		AuthType:       "bearer",
		AuthEnvVar:     "GITHUB_TOKEN",
		MinMintVersion: "0.5.0",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got RegistryEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != entry.Name {
		t.Errorf("Name = %q, want %q", got.Name, entry.Name)
	}
	if got.Description != entry.Description {
		t.Errorf("Description = %q, want %q", got.Description, entry.Description)
	}
	if len(got.Tags) != len(entry.Tags) {
		t.Fatalf("Tags len = %d, want %d", len(got.Tags), len(entry.Tags))
	}
	for i, tag := range got.Tags {
		if tag != entry.Tags[i] {
			t.Errorf("Tags[%d] = %q, want %q", i, tag, entry.Tags[i])
		}
	}
	if got.SpecURL != entry.SpecURL {
		t.Errorf("SpecURL = %q, want %q", got.SpecURL, entry.SpecURL)
	}
	if got.AuthType != entry.AuthType {
		t.Errorf("AuthType = %q, want %q", got.AuthType, entry.AuthType)
	}
	if got.AuthEnvVar != entry.AuthEnvVar {
		t.Errorf("AuthEnvVar = %q, want %q", got.AuthEnvVar, entry.AuthEnvVar)
	}
	if got.MinMintVersion != entry.MinMintVersion {
		t.Errorf("MinMintVersion = %q, want %q", got.MinMintVersion, entry.MinMintVersion)
	}
}

func TestRegistryEntry_JSONFields(t *testing.T) {
	entry := RegistryEntry{
		Name:           "test",
		Description:    "desc",
		Tags:           []string{"a"},
		SpecURL:        "https://example.com/spec.yaml",
		AuthType:       "api_key",
		AuthEnvVar:     "API_KEY",
		MinMintVersion: "1.0.0",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}

	expectedKeys := []string{"name", "description", "tags", "spec_url", "auth_type", "auth_env_var", "min_mint_version"}
	for _, key := range expectedKeys {
		if _, ok := m[key]; !ok {
			t.Errorf("missing JSON key %q", key)
		}
	}
}

func TestRegistryIndex_JSONRoundTrip(t *testing.T) {
	index := RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "github", Description: "GitHub API"},
			{Name: "stripe", Description: "Stripe API"},
		},
	}

	data, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got RegistryIndex
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Version != index.Version {
		t.Errorf("Version = %d, want %d", got.Version, index.Version)
	}
	if len(got.Entries) != len(index.Entries) {
		t.Fatalf("Entries len = %d, want %d", len(got.Entries), len(index.Entries))
	}
	for i, entry := range got.Entries {
		if entry.Name != index.Entries[i].Name {
			t.Errorf("Entries[%d].Name = %q, want %q", i, entry.Name, index.Entries[i].Name)
		}
	}
}

func TestRegistryEntry_EmptyTags(t *testing.T) {
	entry := RegistryEntry{Name: "test"}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got RegistryEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Tags != nil {
		t.Errorf("Tags = %v, want nil", got.Tags)
	}
}

func TestRegistryIndex_EmptyEntries(t *testing.T) {
	index := RegistryIndex{Version: 1}
	data, err := json.Marshal(index)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got RegistryIndex
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Entries != nil {
		t.Errorf("Entries = %v, want nil", got.Entries)
	}
}

// goldenIndexJSON is a minimal on-disk registry document in the real
// {schema_version, apis} shape, including a populated verification block. It is
// the always-on assertion that mint parses the canonical on-disk shape and the
// verification record, without depending on the sibling mcp-registry checkout.
const goldenIndexJSON = `{
  "schema_version": 1,
  "apis": [
    {
      "name": "stripe",
      "description": "Stripe payments API",
      "tags": ["fintech", "payments"],
      "spec_url": "https://example.com/stripe.json",
      "auth_type": "bearer",
      "auth_env_var": "STRIPE_API_KEY",
      "min_mint_version": "0.2.0",
      "verification": {
        "tier": "t1",
        "verified_at": "2026-07-03T00:00:00Z",
        "verifier_version": "0.1.0"
      }
    },
    {
      "name": "unverified",
      "description": "An entry with no verification block yet",
      "tags": ["misc"],
      "spec_url": "https://example.com/other.json",
      "auth_type": "none",
      "auth_env_var": "OTHER_TOKEN",
      "min_mint_version": "0.2.0"
    }
  ]
}`

// TestRegistryIndex_ParsesOnDiskShape proves mint parses the canonical on-disk
// document ({schema_version, apis}) rather than the legacy {version, entries}.
// Before the tag fix this unmarshaled to Version=0, Entries=nil.
func TestRegistryIndex_ParsesOnDiskShape(t *testing.T) {
	var idx RegistryIndex
	if err := json.Unmarshal([]byte(goldenIndexJSON), &idx); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}
	if idx.Version != 1 {
		t.Errorf("Version = %d, want 1 (schema_version not mapped)", idx.Version)
	}
	if len(idx.Entries) != 2 {
		t.Fatalf("Entries len = %d, want 2 (apis not mapped)", len(idx.Entries))
	}
	if idx.Entries[0].Name != "stripe" {
		t.Errorf("Entries[0].Name = %q, want %q", idx.Entries[0].Name, "stripe")
	}
}

// TestRegistryEntry_Verification checks the optional verification block: present
// entries expose the full record, absent ones leave it nil.
func TestRegistryEntry_Verification(t *testing.T) {
	var idx RegistryIndex
	if err := json.Unmarshal([]byte(goldenIndexJSON), &idx); err != nil {
		t.Fatalf("unmarshal golden: %v", err)
	}

	got := idx.Entries[0].Verification
	if got == nil {
		t.Fatal("Verification = nil, want populated block")
	}
	if got.Tier != "t1" {
		t.Errorf("Tier = %q, want %q", got.Tier, "t1")
	}
	if got.VerifiedAt != "2026-07-03T00:00:00Z" {
		t.Errorf("VerifiedAt = %q, want %q", got.VerifiedAt, "2026-07-03T00:00:00Z")
	}
	if got.VerifierVersion != "0.1.0" {
		t.Errorf("VerifierVersion = %q, want %q", got.VerifierVersion, "0.1.0")
	}

	if idx.Entries[1].Verification != nil {
		t.Errorf("Entries[1].Verification = %v, want nil (optional block absent)", idx.Entries[1].Verification)
	}
}

// TestVerification_OmitEmpty confirms the verification block is omitted entirely
// when absent, so the backfill only adds keys and never rewrites existing entries.
func TestVerification_OmitEmpty(t *testing.T) {
	entry := RegistryEntry{Name: "x", Description: "no verification"}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, ok := m["verification"]; ok {
		t.Errorf("verification key present on empty entry, want omitted")
	}
}

// TestRegistryIndex_ParsesRealRegistry reconciles the on-disk vs struct drift
// against the REAL registry.json in the sibling mcp-registry checkout. It guards
// the zero-entries regression: parsing the real catalog must yield entries.
//
// The sibling repo is not present in mint CI, so the test skips when the file is
// absent; set MINT_REGISTRY_FILE to point at a specific registry.json to force it.
func TestRegistryIndex_ParsesRealRegistry(t *testing.T) {
	path := os.Getenv("MINT_REGISTRY_FILE")
	if path == "" {
		for _, cand := range []string{
			"../../../mcp-registry/registry.json",
			"../../../../mcp-registry/registry.json",
		} {
			if _, err := os.Stat(cand); err == nil {
				path = cand
				break
			}
		}
	}
	if path == "" {
		t.Skip("sibling mcp-registry/registry.json not found; set MINT_REGISTRY_FILE to run")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var idx RegistryIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if idx.Version != 1 {
		t.Errorf("Version = %d, want 1", idx.Version)
	}
	if len(idx.Entries) == 0 {
		t.Fatalf("real registry parsed to zero entries: on-disk/struct drift regressed")
	}
	if idx.Entries[0].Name == "" {
		t.Errorf("first entry has empty name; struct did not bind apis[]")
	}
	t.Logf("parsed %d entries from %s (schema_version=%d)", len(idx.Entries), path, idx.Version)
}
