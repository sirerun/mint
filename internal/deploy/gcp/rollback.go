package gcp

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// RevisionClient abstracts Cloud Run revision operations.
type RevisionClient interface {
	ListRevisions(ctx context.Context, serviceName string) ([]Revision, error)
	UpdateTraffic(ctx context.Context, serviceName string, revisionName string, percent int) error
}

// Revision represents a Cloud Run revision.
type Revision struct {
	Name       string
	CreateTime time.Time
	Active     bool // true if receiving traffic
}

// RollbackResult contains the outcome of a rollback operation.
type RollbackResult struct {
	PreviousRevision string
	CurrentRevision  string
	ServiceName      string
}

// Rollback shifts 100% of traffic to the previous revision of a Cloud Run
// service. It returns an error if fewer than 2 revisions exist.
func Rollback(ctx context.Context, client RevisionClient, projectID, region, serviceName string) (*RollbackResult, error) {
	fullName := ServiceFullName(projectID, region, serviceName)

	revisions, err := client.ListRevisions(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("list revisions for %s: %w", fullName, err)
	}

	if len(revisions) == 0 {
		return nil, errors.New("no revisions found: rollback requires at least 2 revisions")
	}
	if len(revisions) < 2 {
		return nil, errors.New("only 1 revision found: rollback requires at least 2 revisions")
	}

	// Identify the active revision and the previous one.
	activeIdx := -1
	for i, r := range revisions {
		if r.Active {
			activeIdx = i
			break
		}
	}
	if activeIdx < 0 {
		return nil, errors.New("no active revision found")
	}

	// The previous revision is the one immediately after the active one in the
	// list (revisions are expected to be ordered newest-first).
	prevIdx := activeIdx + 1
	if prevIdx >= len(revisions) {
		return nil, errors.New("active revision is the oldest: no previous revision to roll back to")
	}

	target := revisions[prevIdx].Name
	current := revisions[activeIdx].Name

	if err := client.UpdateTraffic(ctx, fullName, target, 100); err != nil {
		return nil, fmt.Errorf("update traffic to revision %s: %w", target, err)
	}

	return &RollbackResult{
		PreviousRevision: target,
		CurrentRevision:  current,
		ServiceName:      fullName,
	}, nil
}
