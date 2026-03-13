package managed

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// CreateSourceTarball creates a gzipped tarball of the source directory and writes it to w.
func CreateSourceTarball(sourceDir string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	sourceDir = filepath.Clean(sourceDir)

	return filepath.WalkDir(sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// Use forward slashes in tar headers.
		rel = filepath.ToSlash(rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = rel

		if d.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if d.IsDir() {
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
}

// uploadSource uploads a tarball to the hosting API via multipart POST and returns the source ID.
// Progress is reported to stderr.
func uploadSource(ctx context.Context, client *httpClient, tarball io.Reader, size int64, stderr io.Writer) (string, error) {
	pr, pw := io.Pipe()

	writer := multipart.NewWriter(pw)

	// Write multipart form in a goroutine since it writes to a pipe.
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("source", "source.tar.gz")
		if err != nil {
			errCh <- err
			return
		}

		written := int64(0)
		buf := make([]byte, 32*1024)
		for {
			n, readErr := tarball.Read(buf)
			if n > 0 {
				if _, writeErr := part.Write(buf[:n]); writeErr != nil {
					errCh <- writeErr
					return
				}
				written += int64(n)
				if size > 0 {
					fmt.Fprintf(stderr, "\ruploading: %d / %d bytes", written, size)
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				errCh <- readErr
				return
			}
		}
		if size > 0 {
			fmt.Fprintln(stderr)
		}

		errCh <- writer.Close()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/sources", pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+client.token)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if writeErr := <-errCh; writeErr != nil {
		return "", fmt.Errorf("writing multipart: %w", writeErr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upload HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Extract source ID from response.
	sourceID := strings.TrimSpace(string(body))
	return sourceID, nil
}
