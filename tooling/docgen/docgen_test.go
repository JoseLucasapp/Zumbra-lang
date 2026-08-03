package docgen

import (
	"strings"
	"testing"
)

func TestExtractPublicAPIAndDocumentation(t *testing.T) {
	source := `/// A point in two dimensions.
pub struct Point { x: i32; y: i32; }
/// Adds values.
pub fct add(a, b) { return a + b; }
var hidden << 1;`
	symbols, err := Extract("geometry.zum", source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) != 2 {
		t.Fatalf("expected 2 public symbols, got %#v", symbols)
	}
	if symbols[0].Description != "A point in two dimensions." || len(symbols[0].Members) != 2 {
		t.Fatalf("unexpected struct docs: %#v", symbols[0])
	}
	document := &Document{Title: "Geometry", Symbols: symbols}
	markdown := Markdown(document)
	if !strings.Contains(markdown, "pub fct add(a, b)") {
		t.Fatalf("missing function signature:\n%s", markdown)
	}
}
