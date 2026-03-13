package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// InstallOptions configures the install command.
type InstallOptions struct {
	Name            string
	OutputDir       string
	AuthEnvOverride string
}

// Install downloads an OpenAPI spec from the registry and prints instructions.
func Install(ctx context.Context, index *RegistryIndex, opts InstallOptions, stderr io.Writer) error {
	var entry *RegistryEntry
	for i := range index.Entries {
		if index.Entries[i].Name == opts.Name {
			entry = &index.Entries[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("entry %q not found in registry", opts.Name)
	}

	_, _ = fmt.Fprintf(stderr, "Fetching spec from %s...\n", entry.SpecURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, entry.SpecURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading spec: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading spec: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = opts.Name
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	specFile := filepath.Join(outputDir, "openapi.yaml")
	if err := os.WriteFile(specFile, data, 0o644); err != nil {
		return fmt.Errorf("writing spec: %w", err)
	}

	_, _ = fmt.Fprint(stderr, FormatPostInstall(*entry, outputDir))
	return nil
}

// FormatPostInstall returns post-install instructions for a registry entry.
func FormatPostInstall(entry RegistryEntry, outputDir string) string {
	var b strings.Builder
	b.WriteString("\nSpec saved to " + filepath.Join(outputDir, "openapi.yaml") + "\n\n")
	b.WriteString("Next steps:\n")
	b.WriteString("  1. Generate the MCP server:\n")
	b.WriteString("       mint mcp generate " + filepath.Join(outputDir, "openapi.yaml") + " -o " + outputDir + "\n")
	if entry.AuthEnvVar != "" {
		b.WriteString("  2. Set the auth environment variable:\n")
		b.WriteString("       export " + entry.AuthEnvVar + "=<your-token>\n")
	}
	b.WriteString("  3. Build and run:\n")
	b.WriteString("       cd " + outputDir + " && go build -o server . && ./server\n")
	return b.String()
}
