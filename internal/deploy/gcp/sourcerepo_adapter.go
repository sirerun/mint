package gcp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/sourcerepo/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SourceRepoAdapter implements SourceRepoClient using the Cloud Source
// Repositories REST API.
type SourceRepoAdapter struct {
	service *sourcerepo.Service
}

var _ SourceRepoClient = (*SourceRepoAdapter)(nil)

// NewSourceRepoAdapter creates a new adapter backed by the Cloud Source
// Repositories REST client.
func NewSourceRepoAdapter(ctx context.Context) (*SourceRepoAdapter, error) {
	log.Println("WARNING: Cloud Source Repositories is deprecated. Consider using --no-source-repo.")
	svc, err := sourcerepo.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating source repo service: %w", err)
	}
	return &SourceRepoAdapter{service: svc}, nil
}

// GetRepo returns the named repository. If the repository does not exist,
// it returns a gRPC NotFound status error.
func (a *SourceRepoAdapter) GetRepo(ctx context.Context, req *GetRepoRequest) (*Repo, error) {
	resp, err := a.service.Projects.Repos.Get(req.Name).Context(ctx).Do()
	if err != nil {
		var apiErr *googleapi.Error
		if ok := isGoogleAPIError(err, &apiErr); ok && apiErr.Code == http.StatusNotFound {
			return nil, status.Errorf(codes.NotFound, "repository %s not found", req.Name)
		}
		return nil, err
	}
	return &Repo{Name: resp.Name}, nil
}

// CreateRepo creates a new repository in the specified project.
func (a *SourceRepoAdapter) CreateRepo(ctx context.Context, req *CreateRepoRequest) (*Repo, error) {
	resp, err := a.service.Projects.Repos.Create(req.Parent, &sourcerepo.Repo{
		Name: req.Repo.Name,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return &Repo{Name: resp.Name}, nil
}

// isGoogleAPIError checks whether err is a *googleapi.Error and, if so,
// assigns it to target. This is a small helper to keep the call sites tidy.
func isGoogleAPIError(err error, target **googleapi.Error) bool {
	if e, ok := err.(*googleapi.Error); ok {
		*target = e
		return true
	}
	return false
}
