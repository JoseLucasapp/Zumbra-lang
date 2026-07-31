package parser

import (
	"strings"
	"testing"
	"zumbra/lexer"
)

func TestPeekErrorIncludesLineAndColumn(t *testing.T) {
	input := `
if true {
	var y << 20;
}
`

	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Fatalf("expected parser errors, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err, "line") && strings.Contains(err, "col") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected line/col information in parser errors, got %+v", p.Errors())
	}
}

func TestNoPrefixParseFunctionErrorIncludesTokenInfo(t *testing.T) {
	input := `var x << ;`

	l := lexer.New(input)
	p := New(l)
	_ = p.ParseProgram()

	if len(p.Errors()) == 0 {
		t.Fatalf("expected parser errors, got none")
	}

	found := false
	for _, err := range p.Errors() {
		if strings.Contains(err, "no prefix parse function") {
			found = true
			if !strings.Contains(err, "line") || !strings.Contains(err, "col") {
				t.Fatalf("expected line/col in prefix error, got %q", err)
			}
		}
	}

	if !found {
		t.Fatalf("expected no prefix parse function error, got %+v", p.Errors())
	}
}
