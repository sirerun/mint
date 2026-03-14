package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/auth"
	"github.com/sirerun/mint/internal/publish"
)

func runPublish(args []string) int {
	fs := flag.NewFlagSet("mint publish", flag.ContinueOnError)
	dir := fs.String("dir", ".", "Project directory containing mint.json")
	registryURL := fs.String("registry", defaultRegistryURL, "Registry API base URL")
	dryRun := fs.Bool("dry-run", false, "Validate manifest without uploading")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	// Read and validate manifest first (even for non-dry-run).
	manifest, err := publish.ReadManifest(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	if *dryRun {
		fmt.Println("Dry run: manifest is valid.")
		fmt.Printf("  Name:        %s\n", manifest.Name)
		fmt.Printf("  Version:     %s\n", manifest.Version)
		fmt.Printf("  Description: %s\n", manifest.Description)
		if manifest.Category != "" {
			fmt.Printf("  Category:    %s\n", manifest.Category)
		}
		return 0
	}

	// Load auth token.
	token, err := auth.LoadToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Printf("Publishing %s@%s...\n", manifest.Name, manifest.Version)

	resp, err := publish.Upload(publish.Options{
		Dir:         *dir,
		RegistryURL: *registryURL,
		Token:       token,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Println("Published successfully!")
	fmt.Printf("  Server ID: %s\n", resp.ServerID)
	fmt.Printf("  Version:   %s\n", resp.Version)
	fmt.Printf("  Checksum:  %s\n", resp.Checksum)
	fmt.Printf("  URL:       https://mintmcp.com/servers/%s\n", resp.ServerID)
	return 0
}
