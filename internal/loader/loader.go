package loader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Result holds a parsed OpenAPI document and any errors encountered.
type Result struct {
	Document libopenapi.Document
	Model    *v3high.Document
	Errors   []Error
}

// Error represents a parse or validation error with location info.
type Error struct {
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Path    string `json:"path,omitempty"`
}

func (e Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.Path, e.Line, e.Column, e.Message)
	}
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// Load reads an OpenAPI spec from a file path, URL, or stdin (pass "-").
func Load(source string) (*Result, error) {
	data, sourcePath, err := readSource(source)
	if err != nil {
		return nil, fmt.Errorf("reading source: %w", err)
	}

	return parse(data, sourcePath)
}

// LoadReader reads an OpenAPI spec from an io.Reader.
func LoadReader(r io.Reader, sourcePath string) (*Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	return parse(data, sourcePath)
}

func readSource(source string) ([]byte, string, error) {
	if source == "-" {
		data, err := io.ReadAll(os.Stdin)
		return data, "<stdin>", err
	}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return fetchURL(source)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return nil, "", fmt.Errorf("reading file %s: %w", source, err)
	}
	return data, source, nil
}

func fetchURL(url string) ([]byte, string, error) {
	resp, err := http.Get(url) //nolint:gosec // user-provided URL is intentional
	if err != nil {
		return nil, "", fmt.Errorf("fetching %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fetching %s: status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading response from %s: %w", url, err)
	}
	return data, url, nil
}

func parse(data []byte, sourcePath string) (*Result, error) {
	config := datamodel.DocumentConfiguration{
		AllowFileReferences:   true,
		AllowRemoteReferences: true,
	}

	doc, err := libopenapi.NewDocumentWithConfiguration(data, &config)
	if err != nil {
		return &Result{
			Errors: []Error{{Message: err.Error(), Path: sourcePath}},
		}, fmt.Errorf("parsing spec: %w", err)
	}

	model, err := doc.BuildV3Model()
	result := &Result{
		Document: doc,
	}

	if err != nil {
		result.Errors = append(result.Errors, Error{
			Message: err.Error(),
			Path:    sourcePath,
		})
	}

	if model != nil {
		result.Model = &model.Model
	}

	if model == nil {
		return result, fmt.Errorf("building model from %s: spec produced no model", sourcePath)
	}

	return result, nil
}
