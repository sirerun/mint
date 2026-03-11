package db

import (
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/sirerun/mint/registry/model"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func createTestPublisher(t *testing.T, d *DB) *model.Publisher {
	t.Helper()
	p := &model.Publisher{
		ID:           "pub-1",
		GithubHandle: "testuser",
		APIKeyHash:   HashAPIKey("test-key"),
	}
	if err := d.CreatePublisher(p); err != nil {
		t.Fatalf("create publisher: %v", err)
	}
	return p
}

func createTestServer(t *testing.T, d *DB, pub *model.Publisher) *model.Server {
	t.Helper()
	s := &model.Server{
		ID:            "srv-1",
		Name:          "test-server",
		Description:   "A test server",
		LatestVersion: "1.0.0",
		PublisherID:   pub.ID,
		Category:      "testing",
	}
	if err := d.CreateServer(s); err != nil {
		t.Fatalf("create server: %v", err)
	}
	return s
}

func TestHashAPIKey(t *testing.T) {
	h1 := HashAPIKey("key1")
	h2 := HashAPIKey("key1")
	h3 := HashAPIKey("key2")

	if h1 != h2 {
		t.Error("same key should produce same hash")
	}
	if h1 == h3 {
		t.Error("different keys should produce different hashes")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64", len(h1))
	}
}

func TestCreatePublisher(t *testing.T) {
	tests := []struct {
		name    string
		pub     *model.Publisher
		wantErr bool
	}{
		{
			name: "valid publisher",
			pub: &model.Publisher{
				ID:           "pub-valid",
				GithubHandle: "validuser",
				APIKeyHash:   HashAPIKey("key"),
			},
		},
		{
			name: "missing id",
			pub: &model.Publisher{
				GithubHandle: "user",
				APIKeyHash:   HashAPIKey("key"),
			},
			wantErr: true,
		},
		{
			name: "missing github handle",
			pub: &model.Publisher{
				ID:         "pub-2",
				APIKeyHash: HashAPIKey("key"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := openTestDB(t)
			err := d.CreatePublisher(tt.pub)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePublisher() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreatePublisher_Duplicate(t *testing.T) {
	d := openTestDB(t)
	createTestPublisher(t, d)

	dup := &model.Publisher{
		ID:           "pub-2",
		GithubHandle: "testuser",
		APIKeyHash:   HashAPIKey("other-key"),
	}
	err := d.CreatePublisher(dup)
	if err == nil {
		t.Error("expected error for duplicate github_handle")
	}
}

func TestGetPublisherByAPIKeyHash(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)

	got, err := d.GetPublisherByAPIKeyHash(pub.APIKeyHash)
	if err != nil {
		t.Fatalf("GetPublisherByAPIKeyHash: %v", err)
	}
	if got.ID != pub.ID {
		t.Errorf("ID = %q, want %q", got.ID, pub.ID)
	}

	_, err = d.GetPublisherByAPIKeyHash("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetPublisherByID(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)

	got, err := d.GetPublisherByID(pub.ID)
	if err != nil {
		t.Fatalf("GetPublisherByID: %v", err)
	}
	if got.GithubHandle != pub.GithubHandle {
		t.Errorf("GithubHandle = %q, want %q", got.GithubHandle, pub.GithubHandle)
	}
}

func TestSetPublisherVerified(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)

	if err := d.SetPublisherVerified(pub.ID, true); err != nil {
		t.Fatalf("SetPublisherVerified: %v", err)
	}

	got, _ := d.GetPublisherByID(pub.ID)
	if !got.Verified {
		t.Error("expected publisher to be verified")
	}

	if err := d.SetPublisherVerified("nonexistent", true); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateServer(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	got, err := d.GetServerByID(srv.ID)
	if err != nil {
		t.Fatalf("GetServerByID: %v", err)
	}
	if got.Name != "test-server" {
		t.Errorf("Name = %q, want %q", got.Name, "test-server")
	}
	if got.Category != "testing" {
		t.Errorf("Category = %q, want %q", got.Category, "testing")
	}
}

func TestCreateServer_DuplicateName(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	createTestServer(t, d, pub)

	dup := &model.Server{
		ID:          "srv-2",
		Name:        "test-server",
		Description: "duplicate",
		PublisherID: pub.ID,
	}
	err := d.CreateServer(dup)
	if err == nil {
		t.Error("expected error for duplicate server name")
	}
}

func TestGetServerByName(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	createTestServer(t, d, pub)

	got, err := d.GetServerByName("test-server")
	if err != nil {
		t.Fatalf("GetServerByName: %v", err)
	}
	if got.ID != "srv-1" {
		t.Errorf("ID = %q, want %q", got.ID, "srv-1")
	}

	_, err = d.GetServerByName("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateServer(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	srv.Description = "Updated description"
	srv.LatestVersion = "2.0.0"
	if err := d.UpdateServer(srv); err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}

	got, _ := d.GetServerByID(srv.ID)
	if got.Description != "Updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "Updated description")
	}
	if got.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", got.LatestVersion, "2.0.0")
	}
}

func TestIncrementDownloads(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	for i := 0; i < 5; i++ {
		if err := d.IncrementDownloads(srv.ID); err != nil {
			t.Fatalf("IncrementDownloads: %v", err)
		}
	}

	got, _ := d.GetServerByID(srv.ID)
	if got.Downloads != 5 {
		t.Errorf("Downloads = %d, want 5", got.Downloads)
	}

	if err := d.IncrementDownloads("nonexistent"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSearchServers(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)

	servers := []struct {
		id, name, desc, category string
	}{
		{"s1", "stripe-server", "Stripe payments API", "payments"},
		{"s2", "github-server", "GitHub API integration", "devtools"},
		{"s3", "slack-server", "Slack messaging API", "communication"},
		{"s4", "paypal-server", "PayPal payments API", "payments"},
	}
	for _, s := range servers {
		srv := &model.Server{
			ID: s.id, Name: s.name, Description: s.desc,
			PublisherID: pub.ID, Category: s.category, LatestVersion: "1.0.0",
		}
		if err := d.CreateServer(srv); err != nil {
			t.Fatalf("create server %s: %v", s.name, err)
		}
	}

	tests := []struct {
		name     string
		query    string
		category string
		sort     string
		wantLen  int
	}{
		{"all", "", "", "", 4},
		{"search by name", "stripe", "", "", 1},
		{"search by description", "payments", "", "", 2},
		{"filter by category", "", "payments", "", 2},
		{"search and filter", "API", "devtools", "", 1},
		{"no results", "nonexistent", "", "", 0},
		{"sort by name", "", "", "name", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := d.SearchServers(tt.query, tt.category, tt.sort, 1, 20)
			if err != nil {
				t.Fatalf("SearchServers: %v", err)
			}
			if len(result.Servers) != tt.wantLen {
				t.Errorf("got %d servers, want %d", len(result.Servers), tt.wantLen)
			}
			if result.Total != tt.wantLen {
				t.Errorf("total = %d, want %d", result.Total, tt.wantLen)
			}
		})
	}
}

func TestSearchServers_Pagination(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)

	for i := 0; i < 25; i++ {
		srv := &model.Server{
			ID: "srv-" + string(rune('a'+i)), Name: "server-" + string(rune('a'+i)),
			Description: "desc", PublisherID: pub.ID, LatestVersion: "1.0.0",
		}
		d.CreateServer(srv)
	}

	result, err := d.SearchServers("", "", "", 1, 10)
	if err != nil {
		t.Fatalf("SearchServers: %v", err)
	}
	if len(result.Servers) != 10 {
		t.Errorf("got %d servers, want 10", len(result.Servers))
	}
	if result.Total != 25 {
		t.Errorf("total = %d, want 25", result.Total)
	}

	result2, _ := d.SearchServers("", "", "", 3, 10)
	if len(result2.Servers) != 5 {
		t.Errorf("page 3: got %d servers, want 5", len(result2.Servers))
	}
}

func TestCreateVersion(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	v := &model.Version{
		ID:           "ver-1",
		ServerID:     srv.ID,
		Version:      "1.0.0",
		ArtifactPath: "/tmp/artifact.tar.gz",
		Checksum:     "abc123",
		Changelog:    "Initial release",
	}
	if err := d.CreateVersion(v); err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}

	got, err := d.GetVersion(srv.ID, "1.0.0")
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if got.Checksum != "abc123" {
		t.Errorf("Checksum = %q, want %q", got.Checksum, "abc123")
	}
}

func TestCreateVersion_Duplicate(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	v1 := &model.Version{ID: "v1", ServerID: srv.ID, Version: "1.0.0", ArtifactPath: "/tmp/a.tar.gz"}
	d.CreateVersion(v1)

	v2 := &model.Version{ID: "v2", ServerID: srv.ID, Version: "1.0.0", ArtifactPath: "/tmp/b.tar.gz"}
	err := d.CreateVersion(v2)
	if err == nil {
		t.Error("expected error for duplicate version")
	}
}

func TestGetLatestVersion(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	v1 := &model.Version{ID: "v1", ServerID: srv.ID, Version: "1.0.0", ArtifactPath: "/tmp/a.tar.gz"}
	d.CreateVersion(v1)
	v2 := &model.Version{ID: "v2", ServerID: srv.ID, Version: "2.0.0", ArtifactPath: "/tmp/b.tar.gz"}
	d.CreateVersion(v2)

	got, err := d.GetLatestVersion(srv.ID)
	if err != nil {
		t.Fatalf("GetLatestVersion: %v", err)
	}
	if got.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", got.Version, "2.0.0")
	}
}

func TestListVersions(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	for _, ver := range []string{"1.0.0", "1.1.0", "2.0.0"} {
		v := &model.Version{ID: "v-" + ver, ServerID: srv.ID, Version: ver, ArtifactPath: "/tmp/" + ver}
		d.CreateVersion(v)
	}

	versions, err := d.ListVersions(srv.ID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 3 {
		t.Errorf("got %d versions, want 3", len(versions))
	}
}

func TestToggleStar(t *testing.T) {
	d := openTestDB(t)
	pub := createTestPublisher(t, d)
	srv := createTestServer(t, d, pub)

	// Add star.
	added, err := d.ToggleStar(pub.ID, srv.ID)
	if err != nil {
		t.Fatalf("ToggleStar (add): %v", err)
	}
	if !added {
		t.Error("expected star to be added")
	}

	count, _ := d.GetStarCount(srv.ID)
	if count != 1 {
		t.Errorf("star count = %d, want 1", count)
	}

	// Remove star.
	added, err = d.ToggleStar(pub.ID, srv.ID)
	if err != nil {
		t.Fatalf("ToggleStar (remove): %v", err)
	}
	if added {
		t.Error("expected star to be removed")
	}

	count, _ = d.GetStarCount(srv.ID)
	if count != 0 {
		t.Errorf("star count = %d, want 0", count)
	}
}

func TestGetStarCount_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetStarCount("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
