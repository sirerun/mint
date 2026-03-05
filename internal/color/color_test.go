package color

import (
	"strings"
	"testing"
)

func TestPrinterEnabled(t *testing.T) {
	p := NewWithColor(true)

	tests := []struct {
		name   string
		fn     func(string) string
		escape string
	}{
		{"Error", p.Error, "\033[31m"},
		{"Warning", p.Warning, "\033[33m"},
		{"Info", p.Info, "\033[34m"},
		{"Bold", p.Bold, "\033[1m"},
		{"Gray", p.Gray, "\033[90m"},
	}
	for _, tt := range tests {
		got := tt.fn("text")
		if !strings.Contains(got, tt.escape) {
			t.Errorf("%s() = %q, want escape %q", tt.name, got, tt.escape)
		}
		if !strings.HasSuffix(got, reset) {
			t.Errorf("%s() should end with reset", tt.name)
		}
	}
}

func TestPrinterDisabled(t *testing.T) {
	p := NewWithColor(false)

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Error", p.Error},
		{"Warning", p.Warning},
		{"Info", p.Info},
		{"Bold", p.Bold},
		{"Gray", p.Gray},
	}
	for _, tt := range tests {
		got := tt.fn("text")
		if got != "text" {
			t.Errorf("%s() = %q, want plain %q", tt.name, got, "text")
		}
	}
}

func TestSeverityLabel(t *testing.T) {
	p := NewWithColor(false)

	tests := []struct {
		severity string
		want     string
	}{
		{"error", "[error]"},
		{"warning", "[warning]"},
		{"info", "[info]"},
		{"other", "[other]"},
	}
	for _, tt := range tests {
		got := p.SeverityLabel(tt.severity)
		if got != tt.want {
			t.Errorf("SeverityLabel(%q) = %q, want %q", tt.severity, got, tt.want)
		}
	}
}

func TestSeverityLabelColored(t *testing.T) {
	p := NewWithColor(true)

	got := p.SeverityLabel("error")
	if !strings.Contains(got, "\033[31m") {
		t.Errorf("error label should contain red escape")
	}
	got = p.SeverityLabel("warning")
	if !strings.Contains(got, "\033[33m") {
		t.Errorf("warning label should contain yellow escape")
	}
	got = p.SeverityLabel("info")
	if !strings.Contains(got, "\033[34m") {
		t.Errorf("info label should contain blue escape")
	}
}
