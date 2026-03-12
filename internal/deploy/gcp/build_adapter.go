package gcp

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/storage"
)

// CloudBuildAdapter implements BuildClient using the Cloud Build and Cloud Storage APIs.
type CloudBuildAdapter struct {
	build   *cloudbuild.Client
	storage *storage.Client
}

// Compile-time interface check.
var _ BuildClient = (*CloudBuildAdapter)(nil)

// NewCloudBuildAdapter creates a new CloudBuildAdapter with the given clients.
func NewCloudBuildAdapter(build *cloudbuild.Client, gcs *storage.Client) *CloudBuildAdapter {
	return &CloudBuildAdapter{build: build, storage: gcs}
}

// NewCloudBuildAdapterFromContext creates a CloudBuildAdapter by initializing
// Cloud Build and Cloud Storage clients using Application Default Credentials.
func NewCloudBuildAdapterFromContext(ctx context.Context) (*CloudBuildAdapter, error) {
	buildClient, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating cloud build client: %w", err)
	}
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		buildClient.Close()
		return nil, fmt.Errorf("creating storage client: %w", err)
	}
	return &CloudBuildAdapter{build: buildClient, storage: gcsClient}, nil
}

// Close releases the underlying clients.
func (a *CloudBuildAdapter) Close() error {
	a.storage.Close()
	return a.build.Close()
}

// CreateBuild archives the source directory, uploads it to GCS, submits a
// Cloud Build request, and waits for the build to complete.
func (a *CloudBuildAdapter) CreateBuild(ctx context.Context, projectID string, config *BuildConfig) (*BuildResult, error) {
	bucket := projectID + "_cloudbuild"
	object := fmt.Sprintf("source/%d.tar.gz", time.Now().UnixNano())

	if err := a.ensureBucket(ctx, bucket, projectID); err != nil {
		return nil, fmt.Errorf("ensure build bucket: %w", err)
	}

	if err := a.uploadSource(ctx, bucket, object, config.SourceDir); err != nil {
		return nil, fmt.Errorf("upload source: %w", err)
	}

	req := &cloudbuildpb.CreateBuildRequest{
		ProjectId: projectID,
		Build: &cloudbuildpb.Build{
			Source: &cloudbuildpb.Source{
				Source: &cloudbuildpb.Source_StorageSource{
					StorageSource: &cloudbuildpb.StorageSource{
						Bucket: bucket,
						Object: object,
					},
				},
			},
			Steps: []*cloudbuildpb.BuildStep{
				{
					Name: "gcr.io/cloud-builders/docker",
					Args: []string{"build", "-t", config.ImageURI, "."},
				},
				{
					Name: "gcr.io/cloud-builders/docker",
					Args: []string{"push", config.ImageURI},
				},
			},
			Images: []string{config.ImageURI},
		},
	}

	op, err := a.build.CreateBuild(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create build: %w", err)
	}

	b, err := op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait for build: %w", err)
	}

	var duration time.Duration
	if b.StartTime != nil && b.FinishTime != nil {
		duration = b.FinishTime.AsTime().Sub(b.StartTime.AsTime())
	}

	return &BuildResult{
		ImageURI: config.ImageURI,
		LogURL:   b.LogUrl,
		Duration: duration,
		Status:   b.Status.String(),
	}, nil
}

// ensureBucket creates the Cloud Build source bucket if it does not exist.
func (a *CloudBuildAdapter) ensureBucket(ctx context.Context, bucket, projectID string) error {
	_, err := a.storage.Bucket(bucket).Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		return a.storage.Bucket(bucket).Create(ctx, projectID, &storage.BucketAttrs{
			Location: "us-central1",
		})
	}
	return err
}

// uploadSource creates a tar.gz archive of srcDir and uploads it to GCS.
func (a *CloudBuildAdapter) uploadSource(ctx context.Context, bucket, object, srcDir string) error {
	w := a.storage.Bucket(bucket).Object(object).NewWriter(ctx)
	w.ContentType = "application/gzip"

	gw := gzip.NewWriter(w)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		// Best-effort close on error path.
		tw.Close()
		gw.Close()
		w.Close()
		return fmt.Errorf("archive source: %w", err)
	}

	if err := tw.Close(); err != nil {
		gw.Close()
		w.Close()
		return fmt.Errorf("close tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		w.Close()
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close gcs writer: %w", err)
	}

	return nil
}
