package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirerun/mint/internal/install"
)

func runInstall(args []string) int {
	fs := flag.NewFlagSet("mint install", flag.ContinueOnError)
	registryURL := fs.String("registry", defaultRegistryURL, "Registry API base URL")
	installDir := fs.String("dir", "", "Install directory (default: ~/.mint/servers)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "error: server name is required")
		fmt.Fprintln(os.Stderr, "\nUsage: mint install <name[@version]>")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  mint install stripe-mcp")
		fmt.Fprintln(os.Stderr, "  mint install stripe-mcp@1.2.0")
		return 1
	}

	name := remaining[0]
	parsedName, parsedVersion := install.ParseNameVersion(name)

	versionStr := ""
	if parsedVersion != "" {
		versionStr = "@" + parsedVersion
	}
	fmt.Printf("Installing %s%s...\n", parsedName, versionStr)

	dest, err := install.Install(install.Options{
		Name:        name,
		RegistryURL: *registryURL,
		InstallDir:  *installDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Printf("Installed %s to %s\n", parsedName, dest)
	return 0
}
