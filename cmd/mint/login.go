package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirerun/mint/internal/auth"
	"github.com/sirerun/mint/internal/deploy/managed"
)

const defaultRegistryURL = "https://mint.sire.run/api/v1"

func runLogin(args []string) int {
	fs := flag.NewFlagSet("mint login", flag.ContinueOnError)
	registryURL := fs.String("registry", defaultRegistryURL, "Registry API base URL")
	githubHandle := fs.String("github", "", "GitHub username (for registry login)")
	token := fs.String("token", "", "API token for managed hosting (reads from stdin if omitted)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	// If --token is provided or --github is not, handle managed hosting login.
	if *token != "" || *githubHandle == "" {
		return runLoginManaged(*token)
	}

	// Register with the registry to get an API key.
	url := strings.TrimRight(*registryURL, "/") + "/publishers/register"
	body := fmt.Sprintf(`{"github_handle":%q}`, *githubHandle)

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to connect to registry: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading response: %v\n", err)
		return 1
	}

	if resp.StatusCode == http.StatusConflict {
		fmt.Fprintln(os.Stderr, "Publisher already registered. Use MINT_API_KEY env var if you have your key,")
		fmt.Fprintln(os.Stderr, "or contact support to reset your API key.")
		return 1
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			fmt.Fprintf(os.Stderr, "error: %s\n", errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "error: registration failed (%d)\n", resp.StatusCode)
		}
		return 1
	}

	var result struct {
		PublisherID string `json:"publisher_id"`
		APIKey      string `json:"api_key"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Fprintf(os.Stderr, "error: parsing response: %v\n", err)
		return 1
	}

	// Save credentials.
	if err := auth.SaveCredentials(result.APIKey); err != nil {
		fmt.Fprintf(os.Stderr, "error: saving credentials: %v\n", err)
		return 1
	}

	fmt.Println("Login successful!")
	fmt.Printf("Publisher ID: %s\n", result.PublisherID)
	fmt.Printf("Credentials saved to ~/.mint/credentials\n")
	fmt.Printf("Logged in at: %s\n", time.Now().Format(time.RFC3339))
	return 0
}

func runLoginManaged(token string) int {
	if token == "" {
		fmt.Fprint(os.Stderr, "Enter API token: ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Fprintln(os.Stderr, "error: failed to read token from stdin")
			return 1
		}
		token = strings.TrimSpace(scanner.Text())
	}

	if token == "" {
		fmt.Fprintln(os.Stderr, "error: token cannot be empty")
		return 1
	}

	if err := managed.SaveToken(token); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	fmt.Println("Login successful! Token saved to ~/.config/mint/credentials")
	return 0
}
