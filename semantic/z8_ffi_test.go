package semantic

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func parseZ8Semantic(t *testing.T, source string) *parser.Parser {
	t.Helper()
	return parser.New(lexer.New(source))
}

func TestExternalCallRequiresUnsafe(t *testing.T) {
	p := parseZ8Semantic(t, `extern "C" { fct add(a: i32, b: i32) -> i32; } add(1i32, 2i32);`)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	_, errs := Analyze(program)
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "must be called inside unsafe") {
		t.Fatalf("expected unsafe diagnostic, got %v", errs)
	}
}

func TestExternalCallInsideUnsafe(t *testing.T) {
	p := parseZ8Semantic(t, `extern "C" { fct add(a: i32, b: i32) -> i32; } unsafe { add(1i32, 2i32); }`)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	_, errs := Analyze(program)
	if len(errs) != 0 {
		t.Fatalf("unexpected diagnostics: %v", errs)
	}
}

func TestPubAndExternAreModuleLevelOnly(t *testing.T) {
	p := parseZ8Semantic(t, `fct outer() { pub const visible << 1; extern "C" { fct answer() -> i32; } }`)
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	_, errs := Analyze(program)
	joined := ""
	for _, err := range errs {
		joined += err.Error() + "\n"
	}
	if !strings.Contains(joined, "pub declarations are allowed only at module level") || !strings.Contains(joined, "extern declarations are allowed only at module level") {
		t.Fatalf("expected module-level diagnostics, got %v", errs)
	}
}
