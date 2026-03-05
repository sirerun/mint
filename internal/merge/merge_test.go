package merge

import (
	"os"
	"testing"
)

func TestMergeNoConflicts(t *testing.T) {
	spec1 := []byte(`openapi: "3.0.3"
info:
  title: Spec1
  version: "1.0"
paths:
  /users:
    get:
      operationId: listUsers
      summary: List users
      responses:
        "200":
          description: OK
`)
	spec2 := []byte(`openapi: "3.0.3"
info:
  title: Spec2
  version: "1.0"
paths:
  /pets:
    get:
      operationId: listPets
      summary: List pets
      responses:
        "200":
          description: OK
`)

	result, err := Specs([][]byte{spec1, spec2}, StrategyFail)
	if err != nil {
		t.Fatalf("Specs() error: %v", err)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(result.Conflicts))
	}
	if len(result.Output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestMergeWithConflictFail(t *testing.T) {
	spec, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	_, mergeErr := Specs([][]byte{spec, spec}, StrategyFail)
	if mergeErr == nil {
		t.Error("expected error for conflicting specs with fail strategy")
	}
}

func TestMergeWithConflictSkip(t *testing.T) {
	spec, err := os.ReadFile("../../testdata/petstore.yaml")
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	result, mergeErr := Specs([][]byte{spec, spec}, StrategySkip)
	if mergeErr != nil {
		t.Fatalf("Specs() error: %v", mergeErr)
	}
	if len(result.Conflicts) == 0 {
		t.Error("expected conflicts to be reported even with skip strategy")
	}
}

func TestMergeTooFewSpecs(t *testing.T) {
	_, err := Specs([][]byte{{}}, StrategyFail)
	if err == nil {
		t.Error("expected error for single spec")
	}
}

func TestConflictString(t *testing.T) {
	c := Conflict{Path: "/pets", Detail: "duplicate"}
	want := "conflict at /pets: duplicate"
	if got := c.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
