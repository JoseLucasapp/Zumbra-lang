package types

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func parseProgram(t *testing.T, input string) *parser.Parser {
	t.Helper()
	l := lexer.New(input)
	return parser.New(l)
}

func checkInput(t *testing.T, input string) []error {
	t.Helper()

	p := parseProgram(t, input)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	c := NewChecker()
	return c.Check(program)
}

func TestCheckValidNumericExpression(t *testing.T) {
	errs := checkInput(t, `
		var x << 10 + 20;
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestCheckInvalidNumericExpression(t *testing.T) {
	errs := checkInput(t, `
		var x << 10 + "abc";
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "invalid operands for +") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestCheckHomogeneousArray(t *testing.T) {
	errs := checkInput(t, `
		var xs << [1, 2, 3];
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestCheckMixedArray(t *testing.T) {
	errs := checkInput(t, `
		var xs << [1, "a", 3];
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "array literal has mixed element types") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestCheckComparisonMismatch(t *testing.T) {
	errs := checkInput(t, `
		var ok << 10 == "10";
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "cannot compare") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}
