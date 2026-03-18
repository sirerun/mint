// Package seed provides batch generation of MCP servers from a catalog of OpenAPI specs.
package seed

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Spec represents a single OpenAPI spec entry in the catalog.
type Spec struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	SpecURL     string `json:"spec_url"`
	Description string `json:"description"`
}

// Catalog holds the full list of specs.
type Catalog struct {
	Specs []Spec `json:"specs"`
}

// Result holds the outcome of generating a single server.
type Result struct {
	Name      string        `json:"name"`
	Category  string        `json:"category"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	ToolCount int           `json:"tool_count,omitempty"`
	OutputDir string        `json:"output_dir,omitempty"`
}

// Report summarizes the full batch generation run.
type Report struct {
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
	Duration  time.Duration `json:"duration"`
	Results   []Result      `json:"results"`
}

// Options controls the batch generation behavior.
type Options struct {
	CatalogPath string // path to catalog.json
	OutputDir   string // base output directory
	MintBinary  string // path to the mint binary (default: "mint")
	DryRun      bool   // just validate catalog, don't generate
}

// LoadCatalog reads and parses a catalog JSON file.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read catalog: %w", err)
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parse catalog: %w", err)
	}
	if len(cat.Specs) == 0 {
		return nil, fmt.Errorf("catalog is empty")
	}
	return &cat, nil
}

// ValidateCatalog checks the catalog for common issues.
func ValidateCatalog(cat *Catalog) []string {
	var issues []string
	seen := make(map[string]bool)
	for i, s := range cat.Specs {
		if s.Name == "" {
			issues = append(issues, fmt.Sprintf("spec[%d]: name is required", i))
		}
		if s.SpecURL == "" {
			issues = append(issues, fmt.Sprintf("spec[%d] (%s): spec_url is required", i, s.Name))
		}
		if s.Category == "" {
			issues = append(issues, fmt.Sprintf("spec[%d] (%s): category is required", i, s.Name))
		}
		if s.Description == "" {
			issues = append(issues, fmt.Sprintf("spec[%d] (%s): description is required", i, s.Name))
		}
		if seen[s.Name] {
			issues = append(issues, fmt.Sprintf("spec[%d]: duplicate name %q", i, s.Name))
		}
		seen[s.Name] = true
	}
	return issues
}

// CategoryCounts returns the count of specs per category.
func CategoryCounts(cat *Catalog) map[string]int {
	counts := make(map[string]int)
	for _, s := range cat.Specs {
		counts[s.Category]++
	}
	return counts
}

// Run executes the batch generation process.
func Run(opts Options) (*Report, error) {
	cat, err := LoadCatalog(opts.CatalogPath)
	if err != nil {
		return nil, err
	}

	issues := ValidateCatalog(cat)
	if len(issues) > 0 {
		return nil, fmt.Errorf("catalog validation failed:\n  %s", strings.Join(issues, "\n  "))
	}

	if opts.MintBinary == "" {
		opts.MintBinary = "mint"
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "generated"
	}

	report := &Report{
		Total: len(cat.Specs),
	}
	start := time.Now()

	if opts.DryRun {
		report.Duration = time.Since(start)
		report.Succeeded = len(cat.Specs)
		for _, s := range cat.Specs {
			report.Results = append(report.Results, Result{
				Name:     s.Name,
				Category: s.Category,
				Success:  true,
			})
		}
		return report, nil
	}

	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	for _, spec := range cat.Specs {
		result := generateOne(opts, spec)
		report.Results = append(report.Results, result)
		if result.Success {
			report.Succeeded++
		} else {
			report.Failed++
		}
	}

	report.Duration = time.Since(start)
	return report, nil
}

// generateOne runs mint mcp generate for a single spec.
func generateOne(opts Options, spec Spec) Result {
	outputDir := filepath.Join(opts.OutputDir, spec.Name+"-mcp")
	start := time.Now()

	cmd := exec.Command(opts.MintBinary, "mcp", "generate",
		"--output", outputDir,
		spec.SpecURL,
	)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg == "" {
			errMsg = err.Error()
		}
		return Result{
			Name:     spec.Name,
			Category: spec.Category,
			Success:  false,
			Error:    errMsg,
			Duration: duration,
		}
	}

	// Parse tool count from output (e.g., "  Tools: 42").
	toolCount := parseToolCount(string(output))

	return Result{
		Name:      spec.Name,
		Category:  spec.Category,
		Success:   true,
		Duration:  duration,
		ToolCount: toolCount,
		OutputDir: outputDir,
	}
}

// parseToolCount extracts the tool count from mint mcp generate output.
func parseToolCount(output string) int {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Tools:") {
			var count int
			_, _ = fmt.Sscanf(strings.TrimPrefix(line, "Tools:"), "%d", &count)
			return count
		}
	}
	return 0
}

// FormatReport produces a human-readable summary of the generation report.
func FormatReport(r *Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Batch Generation Report\n")
	fmt.Fprintf(&b, "=======================\n\n")
	fmt.Fprintf(&b, "Total:     %d\n", r.Total)
	fmt.Fprintf(&b, "Succeeded: %d\n", r.Succeeded)
	fmt.Fprintf(&b, "Failed:    %d\n", r.Failed)
	fmt.Fprintf(&b, "Duration:  %s\n\n", r.Duration.Round(time.Millisecond))

	if r.Failed > 0 {
		fmt.Fprintf(&b, "Failures:\n")
		for _, res := range r.Results {
			if !res.Success {
				fmt.Fprintf(&b, "  - %s [%s]: %s\n", res.Name, res.Category, res.Error)
			}
		}
		fmt.Fprintln(&b)
	}

	// Category breakdown.
	cats := make(map[string]struct{ ok, fail int })
	for _, res := range r.Results {
		c := cats[res.Category]
		if res.Success {
			c.ok++
		} else {
			c.fail++
		}
		cats[res.Category] = c
	}
	fmt.Fprintf(&b, "By Category:\n")
	for cat, c := range cats {
		fmt.Fprintf(&b, "  %-20s %d/%d succeeded\n", cat, c.ok, c.ok+c.fail)
	}

	return b.String()
}
