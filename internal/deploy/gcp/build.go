package gcp

import "context"

// BuildClient builds container images.
type BuildClient interface {
	// BuildImage builds a container image from the source directory and pushes
	// it to the specified image URI. It returns the full image URI with digest.
	BuildImage(ctx context.Context, sourceDir, imageURI string) (string, error)
}
