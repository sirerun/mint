package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StatusClient abstracts operations needed for the status command.
type StatusClient interface {
	GetService(ctx context.Context, name string) (*ServiceStatus, error)
	ListRevisions(ctx context.Context, serviceName string) ([]RevisionStatus, error)
}

// ServiceStatus holds the status of a Cloud Run service.
type ServiceStatus struct {
	Name       string
	URL        string
	Labels     map[string]string
	CreateTime time.Time
	UpdateTime time.Time
}

// RevisionStatus holds the status of a revision.
type RevisionStatus struct {
	Name           string
	CreateTime     time.Time
	TrafficPercent int
	Active         bool
}

// StatusResult is the output of the status command.
type StatusResult struct {
	ServiceName string            `json:"service_name"`
	URL         string            `json:"url"`
	Labels      map[string]string `json:"labels"`
	Revisions   []RevisionInfo    `json:"revisions"`
}

// RevisionInfo is a revision in the status output.
type RevisionInfo struct {
	Name           string `json:"name"`
	TrafficPercent int    `json:"traffic_percent"`
	Active         bool   `json:"active"`
	CreateTime     string `json:"create_time"`
}

// GetStatus retrieves the status of a Cloud Run service and its revisions.
func GetStatus(ctx context.Context, client StatusClient, projectID, region, serviceName string) (*StatusResult, error) {
	fullName := fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, region, serviceName)

	svc, err := client.GetService(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("getting service %s: %w", serviceName, err)
	}

	revisions, err := client.ListRevisions(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("listing revisions for %s: %w", serviceName, err)
	}

	result := &StatusResult{
		ServiceName: serviceName,
		URL:         svc.URL,
		Labels:      svc.Labels,
		Revisions:   make([]RevisionInfo, len(revisions)),
	}

	for i, rev := range revisions {
		result.Revisions[i] = RevisionInfo{
			Name:           rev.Name,
			TrafficPercent: rev.TrafficPercent,
			Active:         rev.Active,
			CreateTime:     rev.CreateTime.Format(time.RFC3339),
		}
	}

	return result, nil
}

// FormatStatus formats a StatusResult as either JSON or human-readable text.
func FormatStatus(result *StatusResult, jsonOutput bool) string {
	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		return string(data)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Service:  %s\n", result.ServiceName)
	fmt.Fprintf(&b, "URL:      %s\n", result.URL)

	if len(result.Labels) > 0 {
		fmt.Fprintf(&b, "Labels:\n")
		for k, v := range result.Labels {
			fmt.Fprintf(&b, "  %s=%s\n", k, v)
		}
	}

	if len(result.Revisions) > 0 {
		fmt.Fprintf(&b, "\nRevisions:\n")
		fmt.Fprintf(&b, "  %-40s %-10s %-8s %s\n", "NAME", "TRAFFIC", "ACTIVE", "CREATED")
		for _, rev := range result.Revisions {
			active := "no"
			if rev.Active {
				active = "yes"
			}
			fmt.Fprintf(&b, "  %-40s %-10s %-8s %s\n",
				rev.Name,
				fmt.Sprintf("%d%%", rev.TrafficPercent),
				active,
				rev.CreateTime,
			)
		}
	}

	return b.String()
}
