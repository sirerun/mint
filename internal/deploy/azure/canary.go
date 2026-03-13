package azure

import (
	"context"
	"errors"
	"fmt"
)

// CanaryClient abstracts Container App operations needed for canary traffic splitting.
type CanaryClient interface {
	SetCanaryTraffic(ctx context.Context, serviceName, resourceGroup, canaryRevision string, percent int) error
	PromoteCanary(ctx context.Context, serviceName, resourceGroup, canaryRevision string) error
}

// SetCanaryTraffic routes the given percentage of traffic to the canary revision
// and the remainder to the current active revision. Percent must be between 1 and 99.
func SetCanaryTraffic(ctx context.Context, client ContainerAppClient, resourceGroup, serviceName, canaryRevision string, percent int) error {
	if serviceName == "" {
		return errors.New("service name must not be empty")
	}
	if resourceGroup == "" {
		return errors.New("resource group must not be empty")
	}
	if canaryRevision == "" {
		return errors.New("canary revision must not be empty")
	}
	if percent < 1 || percent > 99 {
		return fmt.Errorf("canary percent must be between 1 and 99, got %d", percent)
	}

	// List revisions to find the current active one.
	revisions, err := client.ListRevisions(ctx, resourceGroup, serviceName)
	if err != nil {
		return fmt.Errorf("list revisions: %w", err)
	}

	// Find the current revision receiving the most traffic.
	var currentRevision string
	maxWeight := -1
	for _, r := range revisions {
		if r.Name != canaryRevision && r.TrafficWeight > maxWeight {
			currentRevision = r.Name
			maxWeight = r.TrafficWeight
		}
	}
	if currentRevision == "" {
		return errors.New("no current revision found to split traffic with")
	}

	stablePercent := 100 - percent
	return client.UpdateTrafficSplit(ctx, resourceGroup, serviceName, []TrafficWeight{
		{RevisionName: currentRevision, Weight: stablePercent},
		{RevisionName: canaryRevision, Weight: percent},
	})
}

// PromoteCanary shifts 100% of traffic to the canary revision, making it the
// new primary.
func PromoteCanary(ctx context.Context, client ContainerAppClient, resourceGroup, serviceName, canaryRevision string) error {
	if serviceName == "" {
		return errors.New("service name must not be empty")
	}
	if resourceGroup == "" {
		return errors.New("resource group must not be empty")
	}
	if canaryRevision == "" {
		return errors.New("canary revision must not be empty")
	}

	return client.UpdateTrafficSplit(ctx, resourceGroup, serviceName, []TrafficWeight{
		{RevisionName: canaryRevision, Weight: 100},
	})
}
