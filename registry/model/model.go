// Package model defines the data types for the Mint registry.
package model

import "time"

// Publisher represents a registered publisher.
type Publisher struct {
	ID           string    `json:"id"`
	GithubHandle string    `json:"github_handle"`
	Verified     bool      `json:"verified"`
	APIKeyHash   string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Server represents a published MCP server package.
type Server struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	LatestVersion  string    `json:"latest_version"`
	OpenAPISpecURL string    `json:"openapi_spec_url,omitempty"`
	PublisherID    string    `json:"publisher_id"`
	Category       string    `json:"category,omitempty"`
	Downloads      int64     `json:"downloads"`
	Stars          int64     `json:"stars"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// Version represents a specific version of a server package.
type Version struct {
	ID           string    `json:"id"`
	ServerID     string    `json:"server_id"`
	Version      string    `json:"version"`
	ArtifactPath string    `json:"artifact_path"`
	Checksum     string    `json:"checksum"`
	Changelog    string    `json:"changelog,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// Star represents a publisher's star on a server.
type Star struct {
	PublisherID string    `json:"publisher_id"`
	ServerID    string    `json:"server_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// PublishRequest is the metadata sent with a publish upload.
type PublishRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Version        string `json:"version"`
	OpenAPISpecURL string `json:"openapi_spec_url,omitempty"`
	Category       string `json:"category,omitempty"`
	Changelog      string `json:"changelog,omitempty"`
}

// ServerListResponse is the paginated response for listing servers.
type ServerListResponse struct {
	Servers  []Server `json:"servers"`
	Total    int      `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// ErrorResponse is returned on errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
