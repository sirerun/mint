package azure

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// RollbackClient abstracts Container App operations needed for rolling back
// to a previous revision.
type RollbackClient interface {
	// ListRevisions returns all revisions for a Container App.
	ListRevisions(ctx context.Context, resourceGroup, appName string) ([]Revision, error)

	// UpdateTrafficSplit updates the traffic distribution across revisions.
	UpdateTrafficSplit(ctx context.Context, resourceGroup, appName string, traffic []TrafficWeight) error
}

// RollbackResult contains the outcome of a rollback operation.
type RollbackResult struct {
	PreviousRevision string
	CurrentRevision  string
	ServiceName      string
	ResourceGroup    string
}

// Rollback shifts 100% of traffic to the previous stable revision of a
// Container App. It returns an error if fewer than 2 revisions exist.
func Rollback(ctx context.Context, client ContainerAppClient, serviceName, resourceGroup string, stderr io.Writer) (*RollbackResult, error) {
	revisions, err := client.ListRevisions(ctx, resourceGroup, serviceName)
	if err != nil {
		return nil, fmt.Errorf("list revisions for %s: %w", serviceName, err)
	}

	if len(revisions) == 0 {
		return nil, errors.New("no revisions found: rollback requires at least 2 revisions")
	}
	if len(revisions) < 2 {
		return nil, errors.New("only 1 revision found: rollback requires at least 2 revisions")
	}

	// Revisions are ordered newest-first. The first is current, the second is
	// the previous stable revision to roll back to.
	current := revisions[0].Name
	previous := revisions[1].Name

	logStderr(stderr, fmt.Sprintf("Rolling back %s from %s to %s...", serviceName, current, previous))

	err = client.UpdateTrafficSplit(ctx, resourceGroup, serviceName, []TrafficWeight{
		{RevisionName: previous, Weight: 100},
	})
	if err != nil {
		return nil, fmt.Errorf("update traffic split for %s: %w", serviceName, err)
	}

	logStderr(stderr, fmt.Sprintf("Rollback complete: %s now receiving 100%% traffic", previous))

	return &RollbackResult{
		PreviousRevision: previous,
		CurrentRevision:  current,
		ServiceName:      serviceName,
		ResourceGroup:    resourceGroup,
	}, nil
}

func logStderr(w io.Writer, msg string) {
	if w != nil {
		_, _ = fmt.Fprintln(w, msg)
	}
}
