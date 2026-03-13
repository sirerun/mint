package registry

import (
	"encoding/json"
	"strings"
	"testing"
)

func searchIndex() *RegistryIndex {
	return &RegistryIndex{
		Version: 1,
		Entries: []RegistryEntry{
			{Name: "github", Description: "GitHub REST API v3", Tags: []string{"scm", "git"}},
			{Name: "stripe", Description: "Stripe payments API", Tags: []string{"payments", "billing"}},
			{Name: "slack", Description: "Slack messaging API", Tags: []string{"messaging", "chat"}},
			{Name: "github-graphql", Description: "GitHub GraphQL API", Tags: []string{"scm", "graphql"}},
			{Name: "twilio", Description: "Twilio SMS and voice", Tags: []string{"messaging", "sms"}},
		},
	}
}

func TestSearch_ExactNameMatch(t *testing.T) {
	results := Search(searchIndex(), "github")
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].Entry.Name != "github" {
		t.Errorf("first result = %q, want github", results[0].Entry.Name)
	}
	if results[0].Score != 1.0 {
		t.Errorf("score = %f, want 1.0", results[0].Score)
	}
}

func TestSearch_NameContains(t *testing.T) {
	results := Search(searchIndex(), "hub")
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	for _, r := range results {
		if !strings.Contains(r.Entry.Name, "hub") {
			t.Errorf("unexpected result: %q", r.Entry.Name)
		}
		if r.Score != 0.8 {
			t.Errorf("score for %q = %f, want 0.8", r.Entry.Name, r.Score)
		}
	}
}

func TestSearch_TagMatch(t *testing.T) {
	results := Search(searchIndex(), "messaging")
	if len(results) == 0 {
		t.Fatal("expected results for tag 'messaging'")
	}
	for _, r := range results {
		if r.Score != 0.6 {
			t.Errorf("score for %q = %f, want 0.6", r.Entry.Name, r.Score)
		}
	}
	names := make(map[string]bool)
	for _, r := range results {
		names[r.Entry.Name] = true
	}
	if !names["slack"] || !names["twilio"] {
		t.Errorf("expected slack and twilio, got %v", names)
	}
}

func TestSearch_DescriptionContains(t *testing.T) {
	results := Search(searchIndex(), "payments")
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	// "payments" is both a tag and in description for stripe.
	// Tag match should win (0.6).
	found := false
	for _, r := range results {
		if r.Entry.Name == "stripe" {
			found = true
			if r.Score != 0.6 {
				t.Errorf("score = %f, want 0.6 (tag match)", r.Score)
			}
		}
	}
	if !found {
		t.Error("stripe not in results")
	}
}

func TestSearch_DescriptionOnly(t *testing.T) {
	results := Search(searchIndex(), "REST")
	if len(results) == 0 {
		t.Fatal("expected results for 'REST'")
	}
	if results[0].Entry.Name != "github" {
		t.Errorf("first result = %q, want github", results[0].Entry.Name)
	}
	if results[0].Score != 0.4 {
		t.Errorf("score = %f, want 0.4", results[0].Score)
	}
}

func TestSearch_NoResults(t *testing.T) {
	results := Search(searchIndex(), "nonexistent")
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	results := Search(searchIndex(), "")
	if results != nil {
		t.Errorf("expected nil for empty query, got %v", results)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	results := Search(searchIndex(), "GITHUB")
	if len(results) == 0 {
		t.Fatal("expected results for case-insensitive match")
	}
	if results[0].Entry.Name != "github" {
		t.Errorf("first result = %q, want github", results[0].Entry.Name)
	}
	if results[0].Score != 1.0 {
		t.Errorf("score = %f, want 1.0", results[0].Score)
	}
}

func TestSearch_ScoringOrder(t *testing.T) {
	// "git" matches: github (name contains=0.8), github-graphql (name contains=0.8),
	// and entries with "git" tag (0.6).
	results := Search(searchIndex(), "git")
	if len(results) == 0 {
		t.Fatal("expected results")
	}

	// Verify descending score order.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted: [%d].Score=%f > [%d].Score=%f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

func TestFormatSearchResults_Table(t *testing.T) {
	results := []SearchResult{
		{Entry: RegistryEntry{Name: "github", Description: "GitHub API"}, Score: 1.0},
		{Entry: RegistryEntry{Name: "gitlab", Description: "GitLab API"}, Score: 0.8},
	}

	out := FormatSearchResults(results, false)
	if !strings.Contains(out, "NAME") {
		t.Error("table missing NAME header")
	}
	if !strings.Contains(out, "github") {
		t.Error("table missing github entry")
	}
	if !strings.Contains(out, "gitlab") {
		t.Error("table missing gitlab entry")
	}
}

func TestFormatSearchResults_TableTruncation(t *testing.T) {
	longDesc := strings.Repeat("a", 60)
	results := []SearchResult{
		{Entry: RegistryEntry{Name: "test", Description: longDesc}, Score: 1.0},
	}

	out := FormatSearchResults(results, false)
	if strings.Contains(out, longDesc) {
		t.Error("long description should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("truncated description should end with ...")
	}
}

func TestFormatSearchResults_EmptyTable(t *testing.T) {
	out := FormatSearchResults(nil, false)
	if out != "No results found." {
		t.Errorf("empty table = %q, want 'No results found.'", out)
	}
}

func TestFormatSearchResults_JSON(t *testing.T) {
	results := []SearchResult{
		{Entry: RegistryEntry{Name: "github", Description: "GitHub API"}, Score: 1.0},
	}

	out := FormatSearchResults(results, true)

	var parsed []SearchResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("len = %d, want 1", len(parsed))
	}
	if parsed[0].Entry.Name != "github" {
		t.Errorf("Name = %q, want github", parsed[0].Entry.Name)
	}
}

func TestFormatSearchResults_JSONEmpty(t *testing.T) {
	out := FormatSearchResults(nil, true)

	var parsed []SearchResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed != nil {
		t.Errorf("expected null, got %v", parsed)
	}
}
