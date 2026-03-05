package overlay

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	data := []byte(`overlay: "1.0.0"
info:
  title: Test Overlay
  version: "1.0"
actions:
  - target: $.info.title
    update: "New Title"
  - target: $.info.x-internal
    remove: true
`)
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if doc.Overlay != "1.0.0" {
		t.Errorf("Overlay = %q, want 1.0.0", doc.Overlay)
	}
	if len(doc.Actions) != 2 {
		t.Fatalf("len(Actions) = %d, want 2", len(doc.Actions))
	}
	if doc.Actions[0].Target != "$.info.title" {
		t.Errorf("action 0 target = %q", doc.Actions[0].Target)
	}
	if doc.Actions[1].Remove != true {
		t.Error("action 1 should be remove")
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse([]byte("not: an: overlay"))
	if err == nil {
		t.Error("expected error for missing overlay field")
	}
}

func TestApplyUpdate(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
info:
  title: Old Title
  version: "1.0"
`)
	overlay := &Document{
		Overlay: "1.0.0",
		Actions: []Action{
			{Target: "$.info.title", Update: "New Title"},
		},
	}

	result, err := Apply(spec, overlay)
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if !strings.Contains(string(result), "New Title") {
		t.Errorf("result should contain 'New Title', got:\n%s", result)
	}
}

func TestApplyRemove(t *testing.T) {
	spec := []byte(`openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
  x-internal: true
`)
	overlay := &Document{
		Overlay: "1.0.0",
		Actions: []Action{
			{Target: "$.info.x-internal", Remove: true},
		},
	}

	result, err := Apply(spec, overlay)
	if err != nil {
		t.Fatalf("Apply() error: %v", err)
	}
	if strings.Contains(string(result), "x-internal") {
		t.Errorf("result should not contain 'x-internal', got:\n%s", result)
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"$.info.title", []string{"info", "title"}},
		{"$.info", []string{"info"}},
		{"$", nil},
		{"$.paths./pets.get", []string{"paths", "/pets", "get"}},
	}
	for _, tt := range tests {
		got := parsePath(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parsePath(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parsePath(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}
