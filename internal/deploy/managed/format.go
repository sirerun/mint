package managed

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatStatus formats a ServerStatus as a human-readable table or JSON string.
func FormatStatus(status *ServerStatus, jsonOutput bool) string {
	if jsonOutput {
		data, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Sprintf("error formatting JSON: %v", err)
		}
		return string(data)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Service ID:  %s\n", status.ServiceID)
	fmt.Fprintf(&b, "URL:         %s\n", status.URL)
	fmt.Fprintf(&b, "State:       %s\n", status.State)
	fmt.Fprintf(&b, "Created:     %s\n", status.CreatedAt.Format("2006-01-02 15:04:05 UTC"))

	if len(status.Revisions) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "REVISION       STATE     TRAFFIC")
		for _, rev := range status.Revisions {
			fmt.Fprintf(&b, "%-14s %-9s %d%%\n", rev.Name, rev.State, rev.TrafficPercent)
		}
	}

	return b.String()
}

// FormatServerList formats a list of ServerSummary entries as a human-readable table or JSON string.
func FormatServerList(servers []ServerSummary, jsonOutput bool) string {
	if jsonOutput {
		data, err := json.MarshalIndent(servers, "", "  ")
		if err != nil {
			return fmt.Sprintf("error formatting JSON: %v", err)
		}
		return string(data)
	}

	if len(servers) == 0 {
		return "No servers found.\n"
	}

	var b strings.Builder
	fmt.Fprintln(&b, "SERVICE ID   NAME            URL                          STATE")
	for _, s := range servers {
		fmt.Fprintf(&b, "%-12s %-15s %-28s %s\n", s.ServiceID, s.ServiceName, s.URL, s.State)
	}

	return b.String()
}
