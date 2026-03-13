package registry

import (
	"encoding/json"
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
