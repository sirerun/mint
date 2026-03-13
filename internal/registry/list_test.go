package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func listIndex() *RegistryIndex {
	return &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "github", Description: "GitHub REST API v3", AuthType: "bearer", Tags: []string{"scm", "git"}},
			{Name: "stripe", Description: "Stripe payments API", AuthType: "api_key", Tags: []string{"payments", "billing"}},
			{Name: "slack", Description: "Slack messaging API", AuthType: "oauth2", Tags: []string{"messaging", "chat"}},
			{Name: "twilio", Description: "Twilio SMS and voice", AuthType: "basic", Tags: []string{"messaging", "sms"}},
		},
	}
}

func TestList_AllEntries(t *testing.T) {
	idx := listIndex()
	got := List(idx, "")
	if len(got) != len(idx.Entries) {
		t.Errorf("len = %d, want %d", len(got), len(idx.Entries))
	}
}

func TestList_TagFilterMatch(t *testing.T) {
	got := List(listIndex(), "messaging")
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	names := map[string]bool{}
	for _, e := range got {
		names[e.Name] = true
	}
	if !names["slack"] || !names["twilio"] {
		t.Errorf("expected slack and twilio, got %v", names)
	}
}

func TestList_TagFilterCaseInsensitive(t *testing.T) {
	got := List(listIndex(), "SCM")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "github" {
		t.Errorf("Name = %q, want github", got[0].Name)
	}
}

func TestList_TagFilterNoMatch(t *testing.T) {
	got := List(listIndex(), "nonexistent")
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestList_MultipleTags(t *testing.T) {
	// "billing" tag only matches stripe.
	got := List(listIndex(), "billing")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Name != "stripe" {
		t.Errorf("Name = %q, want stripe", got[0].Name)
	}
}

func TestFormatList_Table(t *testing.T) {
	entries := listIndex().Entries[:2]
	out := FormatList(entries, false)
	if !strings.Contains(out, "NAME") {
		t.Error("table missing NAME header")
	}
	if !strings.Contains(out, "AUTH TYPE") {
		t.Error("table missing AUTH TYPE header")
	}
	if !strings.Contains(out, "TAGS") {
		t.Error("table missing TAGS header")
	}
	if !strings.Contains(out, "github") {
		t.Error("table missing github entry")
	}
	if !strings.Contains(out, "stripe") {
		t.Error("table missing stripe entry")
	}
}

func TestFormatList_TableEmpty(t *testing.T) {
	out := FormatList(nil, false)
	if out != "No entries found." {
		t.Errorf("empty table = %q, want 'No entries found.'", out)
	}
}

func TestFormatList_TableTruncation(t *testing.T) {
	entries := []RegistryEntry{
		{Name: "test", Description: strings.Repeat("a", 60), Tags: []string{"t"}},
	}
	out := FormatList(entries, false)
	if strings.Contains(out, strings.Repeat("a", 60)) {
		t.Error("long description should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated description should end with ...")
	}
}

func TestFormatList_JSON(t *testing.T) {
	entries := listIndex().Entries[:1]
	out := FormatList(entries, true)

	var parsed []RegistryEntry
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("len = %d, want 1", len(parsed))
	}
	if parsed[0].Name != "github" {
		t.Errorf("Name = %q, want github", parsed[0].Name)
	}
}

func TestFormatList_JSONEmpty(t *testing.T) {
	out := FormatList(nil, true)

	var parsed []RegistryEntry
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed != nil {
		t.Errorf("expected null, got %v", parsed)
	}
}
