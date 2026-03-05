package color

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	bold   = "\033[1m"
	gray   = "\033[90m"
)

// Printer writes colored output when connected to a TTY.
type Printer struct {
	enabled bool
}

// New returns a Printer that uses color when stdout is a terminal.
func New() *Printer {
	return &Printer{enabled: term.IsTerminal(int(os.Stdout.Fd()))}
}

// NewWithColor returns a Printer with color explicitly enabled or disabled.
func NewWithColor(enabled bool) *Printer {
	return &Printer{enabled: enabled}
}

// Error returns text styled as an error (red).
func (p *Printer) Error(s string) string {
	if !p.enabled {
		return s
	}
	return red + s + reset
}

// Warning returns text styled as a warning (yellow).
func (p *Printer) Warning(s string) string {
	if !p.enabled {
		return s
	}
	return yellow + s + reset
}

// Info returns text styled as info (blue).
func (p *Printer) Info(s string) string {
	if !p.enabled {
		return s
	}
	return blue + s + reset
}

// Bold returns bold text.
func (p *Printer) Bold(s string) string {
	if !p.enabled {
		return s
	}
	return bold + s + reset
}

// Gray returns gray/dim text.
func (p *Printer) Gray(s string) string {
	if !p.enabled {
		return s
	}
	return gray + s + reset
}

// Severity returns text colored by severity level.
func (p *Printer) Severity(severity, s string) string {
	switch severity {
	case "error":
		return p.Error(s)
	case "warning":
		return p.Warning(s)
	case "info":
		return p.Info(s)
	default:
		return s
	}
}

// SeverityLabel returns a bracketed severity label with color.
func (p *Printer) SeverityLabel(severity string) string {
	label := fmt.Sprintf("[%s]", severity)
	return p.Severity(severity, label)
}
