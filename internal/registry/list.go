package registry

import (
	"encoding/json"
	"fmt"
	"strings"
)

// List returns entries from the index, optionally filtered by tag.
// If tagFilter is non-empty, only entries with a matching tag (case-insensitive) are returned.
func List(index *RegistryIndex, tagFilter string) []RegistryEntry {
	if tagFilter == "" {
		return index.Entries
	}
	filter := strings.ToLower(tagFilter)
	var matched []RegistryEntry
	for _, e := range index.Entries {
		for _, tag := range e.Tags {
			if strings.ToLower(tag) == filter {
				matched = append(matched, e)
				break
			}
		}
	}
	return matched
}

// FormatList formats entries as either a table or JSON.
func FormatList(entries []RegistryEntry, jsonOutput bool) string {
	if jsonOutput {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		return string(data)
	}

	if len(entries) == 0 {
		return "No entries found."
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-30s %-50s %-12s %s\n", "NAME", "DESCRIPTION", "AUTH TYPE", "TAGS")
	fmt.Fprintf(&b, "%-30s %-50s %-12s %s\n",
		strings.Repeat("-", 30), strings.Repeat("-", 50),
		strings.Repeat("-", 12), strings.Repeat("-", 20))
	for _, e := range entries {
		desc := e.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		tags := strings.Join(e.Tags, ", ")
		fmt.Fprintf(&b, "%-30s %-50s %-12s %s\n", e.Name, desc, e.AuthType, tags)
	}
	return b.String()
}
