package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirerun/mint/internal/seed"
)

func runSeed(args []string) int {
	fs := flag.NewFlagSet("mint seed", flag.ContinueOnError)
	catalogPath := fs.String("catalog", "", "Path to catalog.json (default: built-in catalog)")
	outputDir := fs.String("output", "./generated", "Output directory for generated servers")
	mintBinary := fs.String("mint", "", "Path to mint binary (default: self)")
	dryRun := fs.Bool("dry-run", false, "Validate catalog without generating")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	// Default to built-in catalog.
	if *catalogPath == "" {
		// Find catalog.json relative to this binary's source.
		_, thisFile, _, _ := runtime.Caller(0)
		*catalogPath = filepath.Join(filepath.Dir(thisFile), "..", "..", "internal", "seed", "catalog.json")
	}

	// Default to self for mint binary.
	if *mintBinary == "" {
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot find mint binary: %v\n", err)
			return 1
		}
		*mintBinary = self
	}

	cat, err := seed.LoadCatalog(*catalogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	counts := seed.CategoryCounts(cat)
	fmt.Printf("Catalog: %d specs across %d categories\n", len(cat.Specs), len(counts))
	for cat, n := range counts {
		fmt.Printf("  %-20s %d\n", cat, n)
	}
	fmt.Println()

	if *dryRun {
		issues := seed.ValidateCatalog(cat)
		if len(issues) > 0 {
			fmt.Fprintln(os.Stderr, "Validation issues:")
			for _, issue := range issues {
				fmt.Fprintf(os.Stderr, "  - %s\n", issue)
			}
			return 1
		}
		fmt.Println("Catalog is valid. Dry run complete.")
		return 0
	}

	fmt.Printf("Generating %d servers to %s...\n\n", len(cat.Specs), *outputDir)

	report, err := seed.Run(seed.Options{
		CatalogPath: *catalogPath,
		OutputDir:   *outputDir,
		MintBinary:  *mintBinary,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Print(seed.FormatReport(report))

	if report.Failed > 0 {
		return 1
	}
	return 0
}
