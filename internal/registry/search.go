package registry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SearchResult pairs a registry entry with its relevance score.
type SearchResult struct {
	Entry RegistryEntry `json:"entry"`
	Score float64       `json:"score"`
}

// Search performs fuzzy matching against registry entries.
// Scoring: exact name match = 1.0, name contains = 0.8, tag match = 0.6, description contains = 0.4.
// Results are sorted by score descending.
func Search(index *RegistryIndex, query string) []SearchResult {
	if query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var results []SearchResult

	for _, entry := range index.Entries {
		score := scoreEntry(entry, q)
		if score > 0 {
			results = append(results, SearchResult{Entry: entry, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Entry.Name < results[j].Entry.Name
	})

	return results
}

func scoreEntry(entry RegistryEntry, query string) float64 {
	name := strings.ToLower(entry.Name)
	if name == query {
		return 1.0
	}
	if strings.Contains(name, query) {
		return 0.8
	}
	for _, tag := range entry.Tags {
		if strings.ToLower(tag) == query || strings.Contains(strings.ToLower(tag), query) {
			return 0.6
		}
	}
	if strings.Contains(strings.ToLower(entry.Description), query) {
		return 0.4
	}
	return 0
}

// FormatSearchResults formats results as either a table or JSON.
func FormatSearchResults(results []SearchResult, jsonOutput bool) string {
	if jsonOutput {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return string(data)
	}

	if len(results) == 0 {
		return "No results found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-30s %-50s %s\n", "NAME", "DESCRIPTION", "SCORE")
	fmt.Fprintf(&b, "%-30s %-50s %s\n", strings.Repeat("-", 30), strings.Repeat("-", 50), strings.Repeat("-", 5))
	for _, r := range results {
		desc := r.Entry.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(&b, "%-30s %-50s %.1f\n", r.Entry.Name, desc, r.Score)
	}
	return b.String()
}
