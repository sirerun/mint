package golang

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

	"github.com/sirerun/mint/internal/mcpgen"
)

//go:embed templates/*
var templateFS embed.FS

// templateData extends MCPServer with extra fields needed by templates.
type templateData struct {
	*mcpgen.MCPServer
	ModulePath string
}

// Generate produces a Go MCP server project in the given output directory.
func Generate(server *mcpgen.MCPServer, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	data := &templateData{
		MCPServer:  server,
		ModulePath: server.Name,
	}

	// If no auth, provide a default so templates don't panic
	if data.Auth == nil {
		data.Auth = &mcpgen.MCPAuth{
			Type:       "none",
			HeaderName: "",
			EnvVar:     "MINT_API_KEY",
		}
	}

	funcMap := template.FuncMap{
		"exportName":    exportName,
		"hasBodyParams": hasBodyParams,
	}

	templates := []struct {
		tmpl   string
		output string
	}{
		{"templates/main.go.tmpl", "main.go"},
		{"templates/server.go.tmpl", "server.go"},
		{"templates/tools.go.tmpl", "tools.go"},
		{"templates/client.go.tmpl", "client.go"},
		{"templates/go.mod.tmpl", "go.mod"},
		{"templates/README.md.tmpl", "README.md"},
	}

	for _, t := range templates {
		content, err := templateFS.ReadFile(t.tmpl)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", t.tmpl, err)
		}

		tmpl, err := template.New(t.tmpl).Funcs(funcMap).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", t.tmpl, err)
		}

		outPath := filepath.Join(outputDir, t.output)
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", outPath, err)
		}

		if err := tmpl.Execute(f, data); err != nil {
			_ = f.Close()
			return fmt.Errorf("executing template %s: %w", t.tmpl, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing %s: %w", outPath, err)
		}
	}

	return nil
}

// exportName converts a snake_case name to an exported Go identifier.
// e.g., "list_pets" -> "ListPets"
func exportName(name string) string {
	var result strings.Builder
	upper := true
	for _, r := range name {
		if r == '_' {
			upper = true
			continue
		}
		if upper {
			result.WriteRune(unicode.ToUpper(r))
			upper = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// hasBodyParams checks if any params have In == "body".
func hasBodyParams(params []mcpgen.MCPToolParam) bool {
	for _, p := range params {
		if p.In == "body" {
			return true
		}
	}
	return false
}
