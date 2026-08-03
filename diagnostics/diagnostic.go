// Package diagnostics defines the stable machine-readable diagnostic format
// shared by the compiler pipeline, linter, editor integration and CLI.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
	SeverityHint    Severity = "hint"
)

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	File     string            `json:"file,omitempty"`
	Range    Range             `json:"range"`
	Severity Severity          `json:"severity"`
	Code     string            `json:"code"`
	Source   string            `json:"source"`
	Message  string            `json:"message"`
	Help     string            `json:"help,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func New(file, code, source, message string, severity Severity) Diagnostic {
	line, column := ExtractPosition(message)
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	return Diagnostic{
		File: file,
		Range: Range{
			Start: Position{Line: line, Column: column},
			End:   Position{Line: line, Column: column + 1},
		},
		Severity: severity,
		Code:     code,
		Source:   source,
		Message:  strings.TrimSpace(message),
	}
}

var positionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bat line\s+(\d+)\s*,\s*col(?:umn)?\s+(\d+)`),
	regexp.MustCompile(`(?i)<[^>]*:(\d+):(\d+)>`),
	regexp.MustCompile(`(?i)<(\d+):(\d+)>`),
	regexp.MustCompile(`(?i)\bline\s+(\d+)\s*[:,]\s*(?:col(?:umn)?\s*)?(\d+)`),
}

// ExtractPosition recognizes the position forms emitted by all Zumbra stages.
func ExtractPosition(message string) (int, int) {
	for _, pattern := range positionPatterns {
		match := pattern.FindStringSubmatch(message)
		if len(match) != 3 {
			continue
		}
		line, lineErr := strconv.Atoi(match[1])
		column, columnErr := strconv.Atoi(match[2])
		if lineErr == nil && columnErr == nil {
			return line, column
		}
	}
	return 0, 0
}

func (d Diagnostic) String() string {
	location := d.File
	if d.Range.Start.Line > 0 {
		if location != "" {
			location += ":"
		}
		location += fmt.Sprintf("%d:%d", d.Range.Start.Line, d.Range.Start.Column)
	}
	if location != "" {
		location += ": "
	}
	return fmt.Sprintf("%s%s[%s] %s: %s", location, d.Severity, d.Code, d.Source, d.Message)
}

func Marshal(items []Diagnostic) ([]byte, error) {
	return json.MarshalIndent(items, "", "  ")
}
