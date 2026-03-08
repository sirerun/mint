package gcp

import (
	"context"
	"errors"
	"fmt"
)

// TrafficClient abstracts Cloud Run traffic management.
type TrafficClient interface {
	GetTraffic(ctx context.Context, serviceName string) ([]TrafficTarget, error)
	SetTraffic(ctx context.Context, serviceName string, targets []TrafficTarget) error
}

// TrafficTarget represents a traffic allocation to a revision.
type TrafficTarget struct {
	RevisionName string
	Percent      int
	Tag          string // optional tag like "canary"
}

// CanaryConfig describes a canary deployment.
type CanaryConfig struct {
	ServiceName   string // full resource name
	NewRevision   string
	CanaryPercent int // 1-99
}

// CanaryResult describes the outcome.
type CanaryResult struct {
	NewRevision    string
	NewPercent     int
	StableRevision string
	StablePercent  int
}

// SetCanaryTraffic configures a canary traffic split between the new revision
// and the current stable revision. The canary receives CanaryPercent of traffic
// and the stable revision receives the remainder.
func SetCanaryTraffic(ctx context.Context, client TrafficClient, config CanaryConfig) (*CanaryResult, error) {
	if config.ServiceName == "" {
		return nil, errors.New("service name must not be empty")
	}
	if config.NewRevision == "" {
		return nil, errors.New("new revision must not be empty")
	}
	if config.CanaryPercent < 1 || config.CanaryPercent > 99 {
		return nil, fmt.Errorf("canary percent must be between 1 and 99, got %d", config.CanaryPercent)
	}

	targets, err := client.GetTraffic(ctx, config.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("get current traffic: %w", err)
	}
	if len(targets) == 0 {
		return nil, errors.New("no current traffic targets found")
	}

	// Find the stable revision: the one with the highest traffic percentage.
	stable := stableRevision(targets)

	stablePercent := 100 - config.CanaryPercent
	err = client.SetTraffic(ctx, config.ServiceName, []TrafficTarget{
		{RevisionName: config.NewRevision, Percent: config.CanaryPercent, Tag: "canary"},
		{RevisionName: stable.RevisionName, Percent: stablePercent},
	})
	if err != nil {
		return nil, fmt.Errorf("set canary traffic: %w", err)
	}

	return &CanaryResult{
		NewRevision:    config.NewRevision,
		NewPercent:     config.CanaryPercent,
		StableRevision: stable.RevisionName,
		StablePercent:  stablePercent,
	}, nil
}

// PromoteCanary shifts 100% of traffic to the canary revision. It identifies
// the canary as the revision tagged "canary" or, failing that, the revision
// receiving less than 100% of traffic.
func PromoteCanary(ctx context.Context, client TrafficClient, serviceName string) error {
	if serviceName == "" {
		return errors.New("service name must not be empty")
	}

	targets, err := client.GetTraffic(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("get current traffic: %w", err)
	}
	if len(targets) == 0 {
		return errors.New("no current traffic targets found")
	}

	canary := findCanary(targets)
	if canary == "" {
		return errors.New("no canary revision found")
	}

	err = client.SetTraffic(ctx, serviceName, []TrafficTarget{
		{RevisionName: canary, Percent: 100},
	})
	if err != nil {
		return fmt.Errorf("promote canary: %w", err)
	}

	return nil
}

// stableRevision returns the target with the highest traffic percentage.
func stableRevision(targets []TrafficTarget) TrafficTarget {
	best := targets[0]
	for _, t := range targets[1:] {
		if t.Percent > best.Percent {
			best = t
		}
	}
	return best
}

// findCanary returns the revision name of the canary. It first looks for a
// target tagged "canary", then falls back to the non-100% revision.
func findCanary(targets []TrafficTarget) string {
	for _, t := range targets {
		if t.Tag == "canary" {
			return t.RevisionName
		}
	}
	for _, t := range targets {
		if t.Percent < 100 && t.Percent > 0 {
			return t.RevisionName
		}
	}
	return ""
}
