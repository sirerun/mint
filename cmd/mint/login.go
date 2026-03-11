package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirerun/mint/internal/auth"
)

const defaultRegistryURL = "https://mint.sire.run/api/v1"

func runLogin(args []string) int {
	fs := flag.NewFlagSet("mint login", flag.ContinueOnError)
	registryURL := fs.String("registry", defaultRegistryURL, "Registry API base URL")
	githubHandle := fs.String("github", "", "GitHub username (required)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	if *githubHandle == "" {
		fmt.Fprintln(os.Stderr, "error: --github flag is required")
		fmt.Fprintln(os.Stderr, "\nUsage: mint login --github <username>")
		return 1
	}

	// Register with the registry to get an API key.
	url := strings.TrimRight(*registryURL, "/") + "/publishers/register"
	body := fmt.Sprintf(`{"github_handle":%q}`, *githubHandle)

	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to connect to registry: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

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
