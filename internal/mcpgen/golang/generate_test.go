package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirerun/mint/internal/loader"
	"github.com/sirerun/mint/internal/mcpgen"
)

func TestGenerate(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Check all expected files exist
	expectedFiles := []string{
		"main.go", "server.go", "tools.go", "client.go", "go.mod", "Dockerfile", "README.md", "cli.go",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(outputDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", f)
		}
	}
}

func TestGenerateDockerfileContent(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("reading Dockerfile: %v", err)
	}
	content := string(data)

	// Split into builder and runtime stages
	stages := strings.SplitN(content, "FROM ", 3)
	if len(stages) < 3 {
		t.Fatal("expected at least two FROM stages in Dockerfile")
	}
	runtimeStage := stages[2]

	if !strings.Contains(content, "gcr.io/distroless/static-debian12") {
		t.Error("Dockerfile should contain gcr.io/distroless/static-debian12")
	}

	if !strings.Contains(content, "USER nonroot:nonroot") {
		t.Error("Dockerfile should contain USER nonroot:nonroot")
	}

	if strings.Contains(runtimeStage, "alpine") {
		t.Error("runtime stage should not contain alpine")
	}

	if strings.Contains(content, "apk add") {
		t.Error("Dockerfile should not contain apk add")
	}
}

func TestGenerateCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compilation test in short mode")
	}

	// Check go is available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not found in PATH")
	}

	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	// Run go build
	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}

func TestGenerateGracefulShutdown(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	srv, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(srv, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Check main.go has signal handling
	mainData, err := os.ReadFile(filepath.Join(outputDir, "main.go"))
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	mainContent := string(mainData)
	mainChecks := []string{
		"signal.NotifyContext",
		"syscall.SIGTERM",
		"syscall.SIGINT",
		"SHUTDOWN_TIMEOUT_SECONDS",
		"shutdownTimeout",
	}
	for _, check := range mainChecks {
		if !strings.Contains(mainContent, check) {
			t.Errorf("main.go missing expected string %q", check)
		}
	}

	// Check server.go has graceful shutdown
	serverData, err := os.ReadFile(filepath.Join(outputDir, "server.go"))
	if err != nil {
		t.Fatalf("reading server.go: %v", err)
	}
	serverContent := string(serverData)
	serverChecks := []string{
		"http.Server",
		"ReadTimeout",
		"WriteTimeout",
		"IdleTimeout",
		"httpSrv.Shutdown",
		"shutting down",
		"http.ErrServerClosed",
	}
	for _, check := range serverChecks {
		if !strings.Contains(serverContent, check) {
			t.Errorf("server.go missing expected string %q", check)
		}
	}
}

func TestGenerateHealthEndpoint(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	srv, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(srv, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "server.go"))
	if err != nil {
		t.Fatalf("reading server.go: %v", err)
	}

	content := string(data)
	checks := []string{
		"/health",
		`"status"`,
		"HealthHandler",
		"application/json",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("server.go missing expected string %q", check)
		}
	}
}

func TestGenerateE2E_TwitterAPIv2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not found in PATH")
	}

	// Load the real Twitter/X API v2 OpenAPI spec (754KB, 156 operations).
	result, err := loader.Load("../../../testdata/twitter-v2.json")
	if err != nil {
		t.Fatalf("loading twitter spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting twitter spec: %v", err)
	}

	if len(server.Tools) < 100 {
		t.Errorf("expected 100+ tools from Twitter spec, got %d", len(server.Tools))
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify all expected files exist.
	for _, f := range []string{"main.go", "server.go", "tools.go", "client.go", "go.mod", "Dockerfile"} {
		if _, err := os.Stat(filepath.Join(outputDir, f)); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", f)
		}
	}

	// Verify generated files are non-trivial in size (Twitter spec produces large output).
	for _, f := range []string{"server.go", "tools.go"} {
		info, err := os.Stat(filepath.Join(outputDir, f))
		if err != nil {
			t.Fatalf("stat %s: %v", f, err)
		}
		if info.Size() < 10000 {
			t.Errorf("%s is only %d bytes; expected large output for 156-operation spec", f, info.Size())
		}
	}

	// go mod tidy + go build to verify the generated code compiles.
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	cmd = exec.Command("go", "build", "-buildvcs=false", "./...")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}

func TestExportName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"list_pets", "ListPets"},
		{"create_pet", "CreatePet"},
		{"show_pet_by_id", "ShowPetById"},
		{"simple", "Simple"},
		{"a_b_c", "ABC"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := exportName(tt.input); got != tt.want {
				t.Errorf("exportName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateCLITemplate(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify cli.go exists and contains expected function signatures
	data, err := os.ReadFile(filepath.Join(outputDir, "cli.go"))
	if err != nil {
		t.Fatalf("reading cli.go: %v", err)
	}
	content := string(data)

	checks := []string{
		"func (s *Server) RunCLI(args []string) error",
		"func (s *Server) ListTools(w io.Writer)",
		"func (s *Server) CallTool(ctx context.Context, name string, args map[string]interface{}, raw bool, w io.Writer) error",
		"tabwriter.NewWriter",
		"list_pets",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("cli.go missing expected string %q", check)
		}
	}

	// Verify main.go contains subcommand dispatch
	mainData, err := os.ReadFile(filepath.Join(outputDir, "main.go"))
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	mainContent := string(mainData)

	mainChecks := []string{
		`case "tools"`,
		`case "call"`,
		"srv.ListTools(os.Stdout)",
		"srv.RunCLI(os.Args[2:])",
	}
	for _, check := range mainChecks {
		if !strings.Contains(mainContent, check) {
			t.Errorf("main.go missing expected CLI dispatch string %q", check)
		}
	}
}

func TestGenerateCLIReadme(t *testing.T) {
	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	if err != nil {
		t.Fatalf("reading README.md: %v", err)
	}
	content := string(data)

	checks := []string{
		"## CLI Mode",
		"tools",
		"call <tool_name> key=value",
		"--raw",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("README.md missing expected CLI section string %q", check)
		}
	}
}

func TestGenerateCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not found in PATH")
	}

	result, err := loader.Load("../../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	server, err := mcpgen.Convert(result.Model)
	if err != nil {
		t.Fatalf("converting spec: %v", err)
	}

	outputDir := t.TempDir()

	if err := Generate(server, outputDir); err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Build the binary
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %v\n%s", err, out)
	}

	binaryPath := filepath.Join(outputDir, "server")
	cmd = exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = outputDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	// Test: ./server tools
	cmd = exec.Command(binaryPath, "tools")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("./server tools failed: %v\n%s", err, out)
	}
	toolsOutput := string(out)
	if !strings.Contains(toolsOutput, "list_pets") {
		t.Errorf("tools output missing list_pets: %s", toolsOutput)
	}
	if !strings.Contains(toolsOutput, "TOOL") {
		t.Errorf("tools output missing header: %s", toolsOutput)
	}

	// Test: ./server call nonexistent_tool (should fail with available tools listed)
	cmd = exec.Command(binaryPath, "call", "nonexistent_tool")
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Error("expected error for unknown tool, got nil")
	}
	unknownOutput := string(out)
	if !strings.Contains(unknownOutput, "unknown tool") {
		t.Errorf("unknown tool output missing error message: %s", unknownOutput)
	}
	if !strings.Contains(unknownOutput, "list_pets") {
		t.Errorf("unknown tool output should list available tools: %s", unknownOutput)
	}

	// Test: ./server call list_pets limit=10 (will fail at HTTP call, but parsing should work)
	cmd = exec.Command(binaryPath, "call", "list_pets", "limit=10")
	out, err = cmd.CombinedOutput()
	if err == nil {
		// If it somehow succeeds, that's fine too
		return
	}
	callOutput := string(out)
	// The error should be about the HTTP call, not CLI argument parsing
	if strings.Contains(callOutput, "invalid argument") {
		t.Errorf("call should parse limit=10 correctly, got argument error: %s", callOutput)
	}
}

func TestHasBodyParams(t *testing.T) {
	tests := []struct {
		name   string
		params []mcpgen.MCPToolParam
		want   bool
	}{
		{
			name:   "no params",
			params: nil,
			want:   false,
		},
		{
			name:   "query params only",
			params: []mcpgen.MCPToolParam{{In: "query"}},
			want:   false,
		},
		{
			name:   "has body param",
			params: []mcpgen.MCPToolParam{{In: "body"}},
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasBodyParams(tt.params); got != tt.want {
				t.Errorf("hasBodyParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
