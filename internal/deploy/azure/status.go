package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// jsonMarshalIndent is a variable to allow tests to inject marshal failures.
var jsonMarshalIndent = json.MarshalIndent

// StatusClient abstracts operations needed for the Azure Container Apps status command.
type StatusClient interface {
	GetStatus(ctx context.Context, serviceName, resourceGroup string) (*StatusResult, error)
}

// StatusResult holds the status of an Azure Container App.
type StatusResult struct {
	ServiceName       string          `json:"service_name"`
	ResourceGroup     string          `json:"resource_group"`
	FQDN              string          `json:"fqdn"`
	ProvisioningState string          `json:"provisioning_state"`
	LatestRevision    string          `json:"latest_revision"`
	Revisions         []RevisionInfo  `json:"revisions"`
	TrafficWeights    []TrafficWeight `json:"traffic_weights"`
}

// RevisionInfo holds metadata about a Container App revision.
type RevisionInfo struct {
	Name              string `json:"name"`
	CreatedTime       string `json:"created_time"`
	ProvisioningState string `json:"provisioning_state"`
	RunningState      string `json:"running_state"`
	Replicas          int    `json:"replicas"`
}

// GetContainerAppStatus retrieves the status of a Container App including its
// revisions and traffic weights.
func GetContainerAppStatus(ctx context.Context, client ContainerAppClient, resourceGroup, serviceName string) (*StatusResult, error) {
	app, err := client.GetApp(ctx, resourceGroup, serviceName)
	if err != nil {
		return nil, fmt.Errorf("getting app %s: %w", serviceName, err)
	}

	result := &StatusResult{
		ServiceName:       app.Name,
		ResourceGroup:     resourceGroup,
		FQDN:              app.FQDN,
		ProvisioningState: app.ProvisioningState,
		LatestRevision:    app.LatestRevision,
	}

	revisions, err := client.ListRevisions(ctx, resourceGroup, serviceName)
	if err != nil {
		// Return partial result with app info when revision listing fails.
		return result, fmt.Errorf("listing revisions: %w", err)
	}

	result.Revisions = make([]RevisionInfo, len(revisions))
	for i, r := range revisions {
		result.Revisions[i] = RevisionInfo{
			Name:        r.Name,
			CreatedTime: r.CreatedTime,
		}
	}

	result.TrafficWeights = make([]TrafficWeight, 0)
	for _, r := range revisions {
		if r.TrafficWeight > 0 {
			result.TrafficWeights = append(result.TrafficWeights, TrafficWeight{
				RevisionName: r.Name,
				Weight:       r.TrafficWeight,
			})
		}
	}

	return result, nil
}

// FormatStatus formats a StatusResult as either JSON or human-readable text.
func FormatStatus(result *StatusResult, jsonOutput bool) string {
	if jsonOutput {
		data, err := jsonMarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		return string(data)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Service:           %s\n", result.ServiceName)
	fmt.Fprintf(&b, "Resource Group:    %s\n", result.ResourceGroup)
	fmt.Fprintf(&b, "FQDN:              %s\n", result.FQDN)
	fmt.Fprintf(&b, "Provisioning:      %s\n", result.ProvisioningState)
	fmt.Fprintf(&b, "Latest Revision:   %s\n", result.LatestRevision)

	if len(result.Revisions) > 0 {
		fmt.Fprintf(&b, "\nRevisions:\n")
		fmt.Fprintf(&b, "  %-40s %-20s %-16s %-12s %s\n", "NAME", "CREATED", "PROVISIONING", "RUNNING", "REPLICAS")
		for _, r := range result.Revisions {
			fmt.Fprintf(&b, "  %-40s %-20s %-16s %-12s %d\n", r.Name, r.CreatedTime, r.ProvisioningState, r.RunningState, r.Replicas)
		}
	}

	if len(result.TrafficWeights) > 0 {
		fmt.Fprintf(&b, "\nTraffic:\n")
		fmt.Fprintf(&b, "  %-40s %s\n", "REVISION", "WEIGHT")
		for _, tw := range result.TrafficWeights {
			fmt.Fprintf(&b, "  %-40s %d%%\n", tw.RevisionName, tw.Weight)
		}
	}

	return b.String()
}
