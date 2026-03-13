package managed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HostingClient defines the interface for managed hosting operations.
type HostingClient interface {
	Deploy(ctx context.Context, input *DeployInput) (*DeployOutput, error)
	Status(ctx context.Context, serviceID string) (*ServerStatus, error)
	Delete(ctx context.Context, serviceID string) error
	ListServers(ctx context.Context) ([]ServerSummary, error)
}

// DeployInput contains the parameters for deploying a server.
type DeployInput struct {
	Source      string `json:"source"`
	ServiceName string `json:"service_name"`
	Public      bool   `json:"public"`
}

// DeployOutput contains the result of a deployment.
type DeployOutput struct {
	URL       string `json:"url"`
	ServiceID string `json:"service_id"`
	BuildID   string `json:"build_id"`
}

// ServerStatus contains the status of a deployed server.
type ServerStatus struct {
	ServiceID string         `json:"service_id"`
	URL       string         `json:"url"`
	State     string         `json:"state"`
	Revisions []RevisionInfo `json:"revisions"`
	CreatedAt time.Time      `json:"created_at"`
}

// ServerSummary contains a brief summary of a deployed server.
type ServerSummary struct {
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
	URL         string `json:"url"`
	State       string `json:"state"`
}

// RevisionInfo contains information about a server revision.
type RevisionInfo struct {
	Name           string `json:"name"`
	State          string `json:"state"`
	TrafficPercent int    `json:"traffic_percent"`
}

// httpClient implements HostingClient using HTTP requests.
type httpClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new HostingClient targeting the given base URL with the given API token.
func NewClient(baseURL string, token string) HostingClient {
	if baseURL == "" {
		baseURL = "https://api.sire.run/v1/hosting"
	}
	return &httpClient{
		baseURL:    baseURL,
		token:      token,
		httpClient: &http.Client{},
	}
}

func (c *httpClient) doJSON(req *http.Request, result interface{}) error {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func (c *httpClient) Deploy(ctx context.Context, input *DeployInput) (*DeployOutput, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encoding deploy input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/services", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var out DeployOutput
	if err := c.doJSON(req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *httpClient) Status(ctx context.Context, serviceID string) (*ServerStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/services/"+serviceID, nil)
	if err != nil {
		return nil, err
	}

	var status ServerStatus
	if err := c.doJSON(req, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *httpClient) Delete(ctx context.Context, serviceID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/services/"+serviceID, nil)
	if err != nil {
		return err
	}
	return c.doJSON(req, nil)
}

func (c *httpClient) ListServers(ctx context.Context) ([]ServerSummary, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/services", nil)
	if err != nil {
		return nil, err
	}

	var servers []ServerSummary
	if err := c.doJSON(req, &servers); err != nil {
		return nil, err
	}
	return servers, nil
}
