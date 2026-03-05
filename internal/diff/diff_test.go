package diff

import (
	"testing"

	"github.com/sirerun/mint/internal/loader"
)

func loadSpec(t *testing.T, path string) *loader.Result {
	t.Helper()
	result, err := loader.Load(path)
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	return result
}

func TestDiffIdenticalSpecs(t *testing.T) {
	old := loadSpec(t, "../../testdata/petstore.yaml")
	new := loadSpec(t, "../../testdata/petstore.yaml")

	result := Specs(old.Model, new.Model)
	if len(result.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(result.Changes))
		for _, c := range result.Changes {
			t.Logf("  %s", c)
		}
	}
}

func TestDiffBreakingChanges(t *testing.T) {
	old := loadSpec(t, "../../testdata/petstore.yaml")
	// Use the JSON petstore which only has GET /pets (no POST, no /pets/{petId})
	new := loadSpec(t, "../../testdata/petstore.json")

	result := Specs(old.Model, new.Model)
	if !result.HasBreaking {
		t.Error("expected breaking changes")
	}
	if result.BreakingChanges == 0 {
		t.Error("expected non-zero breaking changes count")
	}
}

func TestChangeString(t *testing.T) {
	tests := []struct {
		c    Change
		want string
	}{
		{
			Change{Type: Removed, Path: "/pets", Method: "POST", Detail: "operation removed", Breaking: true},
			"[removed] POST /pets: operation removed [BREAKING]",
		},
		{
			Change{Type: Added, Path: "/users", Detail: "path added"},
			"[added] /users: path added",
		},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}
