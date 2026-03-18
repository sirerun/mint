// Package db provides SQLite storage for the Mint registry.
package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirerun/mint/registry/model"
)

// Common errors.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

// DB wraps a sql.DB with registry operations.
type DB struct {
	conn *sql.DB
}

// Open creates or opens a SQLite database at the given path.
// Use ":memory:" for an in-memory database (useful for tests).
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Enable WAL mode for better concurrent read performance.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS registry_publishers (
			id TEXT PRIMARY KEY,
			github_handle TEXT UNIQUE NOT NULL,
			verified INTEGER NOT NULL DEFAULT 0,
			api_key_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS registry_servers (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			latest_version TEXT NOT NULL DEFAULT '',
			openapi_spec_url TEXT NOT NULL DEFAULT '',
			publisher_id TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			downloads INTEGER NOT NULL DEFAULT 0,
			stars INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (publisher_id) REFERENCES registry_publishers(id)
		)`,
		`CREATE TABLE IF NOT EXISTS registry_versions (
			id TEXT PRIMARY KEY,
			server_id TEXT NOT NULL,
			version TEXT NOT NULL,
			artifact_path TEXT NOT NULL,
			checksum TEXT NOT NULL DEFAULT '',
			changelog TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (server_id) REFERENCES registry_servers(id),
			UNIQUE(server_id, version)
		)`,
		`CREATE TABLE IF NOT EXISTS registry_stars (
			publisher_id TEXT NOT NULL,
			server_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (publisher_id, server_id),
			FOREIGN KEY (publisher_id) REFERENCES registry_publishers(id),
			FOREIGN KEY (server_id) REFERENCES registry_servers(id)
		)`,
	}
	for _, s := range stmts {
		if _, err := d.conn.Exec(s); err != nil {
			return fmt.Errorf("execute %q: %w", s[:40], err)
		}
	}
	return nil
}

// HashAPIKey returns a SHA-256 hex hash of the given API key.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// CreatePublisher inserts a new publisher.
func (d *DB) CreatePublisher(p *model.Publisher) error {
	if p.ID == "" || p.GithubHandle == "" || p.APIKeyHash == "" {
		return fmt.Errorf("%w: publisher id, github_handle, and api_key_hash are required", ErrInvalidInput)
	}
	p.CreatedAt = time.Now().UTC()
	_, err := d.conn.Exec(
		`INSERT INTO registry_publishers (id, github_handle, verified, api_key_hash, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.GithubHandle, boolToInt(p.Verified), p.APIKeyHash, p.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: publisher with that github_handle already exists", ErrAlreadyExists)
		}
		return fmt.Errorf("insert publisher: %w", err)
	}
	return nil
}

// GetPublisherByAPIKeyHash returns the publisher matching the given key hash.
func (d *DB) GetPublisherByAPIKeyHash(hash string) (*model.Publisher, error) {
	row := d.conn.QueryRow(
		`SELECT id, github_handle, verified, api_key_hash, created_at FROM registry_publishers WHERE api_key_hash = ?`,
		hash,
	)
	return scanPublisher(row)
}

// GetPublisherByID returns a publisher by ID.
func (d *DB) GetPublisherByID(id string) (*model.Publisher, error) {
	row := d.conn.QueryRow(
		`SELECT id, github_handle, verified, api_key_hash, created_at FROM registry_publishers WHERE id = ?`,
		id,
	)
	return scanPublisher(row)
}

// SetPublisherVerified sets the verified flag for a publisher.
func (d *DB) SetPublisherVerified(id string, verified bool) error {
	res, err := d.conn.Exec(
		`UPDATE registry_publishers SET verified = ? WHERE id = ?`,
		boolToInt(verified), id,
	)
	if err != nil {
		return fmt.Errorf("update publisher: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanPublisher(row *sql.Row) (*model.Publisher, error) {
	var p model.Publisher
	var verified int
	var createdAt string
	err := row.Scan(&p.ID, &p.GithubHandle, &verified, &p.APIKeyHash, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan publisher: %w", err)
	}
	p.Verified = verified != 0
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &p, nil
}

// CreateServer inserts a new server record.
func (d *DB) CreateServer(s *model.Server) error {
	now := time.Now().UTC()
	s.CreatedAt = now
	s.UpdatedAt = now
	_, err := d.conn.Exec(
		`INSERT INTO registry_servers (id, name, description, latest_version, openapi_spec_url, publisher_id, category, downloads, stars, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.Description, s.LatestVersion, s.OpenAPISpecURL,
		s.PublisherID, s.Category, s.Downloads, s.Stars,
		s.CreatedAt.Format(time.RFC3339), s.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: server name already taken", ErrAlreadyExists)
		}
		return fmt.Errorf("insert server: %w", err)
	}
	return nil
}

// GetServerByID returns a server by ID.
func (d *DB) GetServerByID(id string) (*model.Server, error) {
	row := d.conn.QueryRow(
		`SELECT id, name, description, latest_version, openapi_spec_url, publisher_id, category, downloads, stars, created_at, updated_at
		 FROM registry_servers WHERE id = ?`, id,
	)
	return scanServer(row)
}

// GetServerByName returns a server by name.
func (d *DB) GetServerByName(name string) (*model.Server, error) {
	row := d.conn.QueryRow(
		`SELECT id, name, description, latest_version, openapi_spec_url, publisher_id, category, downloads, stars, created_at, updated_at
		 FROM registry_servers WHERE name = ?`, name,
	)
	return scanServer(row)
}

// UpdateServer updates mutable server fields.
func (d *DB) UpdateServer(s *model.Server) error {
	s.UpdatedAt = time.Now().UTC()
	_, err := d.conn.Exec(
		`UPDATE registry_servers SET description = ?, latest_version = ?, openapi_spec_url = ?, category = ?, updated_at = ? WHERE id = ?`,
		s.Description, s.LatestVersion, s.OpenAPISpecURL, s.Category, s.UpdatedAt.Format(time.RFC3339), s.ID,
	)
	return err
}

// IncrementDownloads atomically increments a server's download count.
func (d *DB) IncrementDownloads(serverID string) error {
	res, err := d.conn.Exec(
		`UPDATE registry_servers SET downloads = downloads + 1 WHERE id = ?`, serverID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SearchServers performs full-text search with optional category filter and pagination.
func (d *DB) SearchServers(query, category, sort string, page, pageSize int) (*model.ServerListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var where []string
	var args []any

	if query != "" {
		where = append(where, "(name LIKE ? OR description LIKE ?)")
		q := "%" + query + "%"
		args = append(args, q, q)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Count total.
	var total int
	countQuery := "SELECT COUNT(*) FROM registry_servers " + whereClause
	if err := d.conn.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count servers: %w", err)
	}

	orderBy := "updated_at DESC"
	switch sort {
	case "downloads":
		orderBy = "downloads DESC"
	case "stars":
		orderBy = "stars DESC"
	case "name":
		orderBy = "name ASC"
	case "recent":
		orderBy = "created_at DESC"
	}

	offset := (page - 1) * pageSize
	selectQuery := fmt.Sprintf(
		`SELECT id, name, description, latest_version, openapi_spec_url, publisher_id, category, downloads, stars, created_at, updated_at
		 FROM registry_servers %s ORDER BY %s LIMIT ? OFFSET ?`,
		whereClause, orderBy,
	)
	args = append(args, pageSize, offset)

	rows, err := d.conn.Query(selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query servers: %w", err)
	}
	defer rows.Close()

	var servers []model.Server
	for rows.Next() {
		var s model.Server
		var createdAt, updatedAt string
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.LatestVersion, &s.OpenAPISpecURL,
			&s.PublisherID, &s.Category, &s.Downloads, &s.Stars, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan server row: %w", err)
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		servers = append(servers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate server rows: %w", err)
	}

	return &model.ServerListResponse{
		Servers:  servers,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func scanServer(row *sql.Row) (*model.Server, error) {
	var s model.Server
	var createdAt, updatedAt string
	err := row.Scan(&s.ID, &s.Name, &s.Description, &s.LatestVersion, &s.OpenAPISpecURL,
		&s.PublisherID, &s.Category, &s.Downloads, &s.Stars, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan server: %w", err)
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &s, nil
}

// CreateVersion inserts a new version record.
func (d *DB) CreateVersion(v *model.Version) error {
	v.CreatedAt = time.Now().UTC()
	_, err := d.conn.Exec(
		`INSERT INTO registry_versions (id, server_id, version, artifact_path, checksum, changelog, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.ServerID, v.Version, v.ArtifactPath, v.Checksum, v.Changelog, v.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("%w: version %s already exists for this server", ErrAlreadyExists, v.Version)
		}
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}

// GetVersion returns a specific version of a server.
func (d *DB) GetVersion(serverID, version string) (*model.Version, error) {
	row := d.conn.QueryRow(
		`SELECT id, server_id, version, artifact_path, checksum, changelog, created_at
		 FROM registry_versions WHERE server_id = ? AND version = ?`,
		serverID, version,
	)
	return scanVersion(row)
}

// GetLatestVersion returns the latest version of a server.
func (d *DB) GetLatestVersion(serverID string) (*model.Version, error) {
	row := d.conn.QueryRow(
		`SELECT id, server_id, version, artifact_path, checksum, changelog, created_at
		 FROM registry_versions WHERE server_id = ? ORDER BY created_at DESC, rowid DESC LIMIT 1`,
		serverID,
	)
	return scanVersion(row)
}

// ListVersions lists all versions of a server.
func (d *DB) ListVersions(serverID string) ([]model.Version, error) {
	rows, err := d.conn.Query(
		`SELECT id, server_id, version, artifact_path, checksum, changelog, created_at
		 FROM registry_versions WHERE server_id = ? ORDER BY created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, fmt.Errorf("query versions: %w", err)
	}
	defer rows.Close()

	var versions []model.Version
	for rows.Next() {
		var v model.Version
		var createdAt string
		if err := rows.Scan(&v.ID, &v.ServerID, &v.Version, &v.ArtifactPath, &v.Checksum, &v.Changelog, &createdAt); err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func scanVersion(row *sql.Row) (*model.Version, error) {
	var v model.Version
	var createdAt string
	err := row.Scan(&v.ID, &v.ServerID, &v.Version, &v.ArtifactPath, &v.Checksum, &v.Changelog, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan version: %w", err)
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &v, nil
}

// ToggleStar adds or removes a star. Returns true if the star was added.
func (d *DB) ToggleStar(publisherID, serverID string) (bool, error) {
	// Check if the star exists.
	var exists int
	err := d.conn.QueryRow(
		`SELECT COUNT(*) FROM registry_stars WHERE publisher_id = ? AND server_id = ?`,
		publisherID, serverID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check star: %w", err)
	}

	if exists > 0 {
		// Remove star.
		if _, err := d.conn.Exec(
			`DELETE FROM registry_stars WHERE publisher_id = ? AND server_id = ?`,
			publisherID, serverID,
		); err != nil {
			return false, fmt.Errorf("delete star: %w", err)
		}
		if _, err := d.conn.Exec(
			`UPDATE registry_servers SET stars = stars - 1 WHERE id = ?`, serverID,
		); err != nil {
			return false, fmt.Errorf("decrement stars: %w", err)
		}
		return false, nil
	}

	// Add star.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := d.conn.Exec(
		`INSERT INTO registry_stars (publisher_id, server_id, created_at) VALUES (?, ?, ?)`,
		publisherID, serverID, now,
	); err != nil {
		return false, fmt.Errorf("insert star: %w", err)
	}
	if _, err := d.conn.Exec(
		`UPDATE registry_servers SET stars = stars + 1 WHERE id = ?`, serverID,
	); err != nil {
		return false, fmt.Errorf("increment stars: %w", err)
	}
	return true, nil
}

// GetStarCount returns the number of stars for a server.
func (d *DB) GetStarCount(serverID string) (int64, error) {
	var count int64
	err := d.conn.QueryRow(`SELECT stars FROM registry_servers WHERE id = ?`, serverID).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return count, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
