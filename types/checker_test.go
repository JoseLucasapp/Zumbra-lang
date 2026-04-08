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

func TestBuiltinSizeOfReturnsInt(t *testing.T) {
	errs := checkInput(t, `
		var len << sizeOf([1, 2, 3]);
		var x << len + 10;
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestBuiltinInputReturnsString(t *testing.T) {
	errs := checkInput(t, `
		var name << input("name");
		var msg << name + "!";
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestFunctionReturnInferenceInt(t *testing.T) {
	errs := checkInput(t, `
		var answer << fct() { return 42; };
		var x << answer() + 8;
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestFunctionReturnInferenceString(t *testing.T) {
	errs := checkInput(t, `
		var greet << fct() { return "hi"; };
		var msg << greet() + "!";
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestFunctionConflictingReturnTypes(t *testing.T) {
	errs := checkInput(t, `
		var strange << fct() {
			return 10;
			return "abc";
		};
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "function has conflicting return types") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestArrayIndexMustBeInt(t *testing.T) {
	errs := checkInput(t, `
		var xs << [1, 2, 3];
		var x << xs["0"];
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "array index must be int") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestIfConditionMustBeBool(t *testing.T) {
	errs := checkInput(t, `
		if ("abc") { show("x"); }
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "if condition must be bool") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestWhileConditionMustBeBool(t *testing.T) {
	errs := checkInput(t, `
		while (123) { show("y"); }
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "while condition must be bool") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestSizeOfRejectsInvalidType(t *testing.T) {
	errs := checkInput(t, `
		var x << sizeOf(10);
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "sizeOf expects array or string") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestBuiltinArityValidation(t *testing.T) {
	errs := checkInput(t, `
		show();
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "show expects 1 argument") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestFirstReturnsElementType(t *testing.T) {
	errs := checkInput(t, `
		var xs << [1, 2, 3];
		var x << first(xs) + 1;
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestUserFunctionArityMismatchTooFew(t *testing.T) {
	errs := checkInput(t, `
		var add << fct(a, b) { return a + b; };
		add(1);
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "function expects 2 arguments, got 1") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestUserFunctionArityMismatchTooMany(t *testing.T) {
	errs := checkInput(t, `
		var add << fct(a, b) { return a + b; };
		add(1, 2, 3);
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "function expects 2 arguments, got 3") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestUserFunctionCorrectArity(t *testing.T) {
	errs := checkInput(t, `
		var add << fct(a, b) { return a + b; };
		add(1, 2);
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestZeroArgFunctionRejectsExtraArgs(t *testing.T) {
	errs := checkInput(t, `
		var answer << fct() { return 42; };
		answer(1);
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "function expects 0 arguments, got 1") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestBuiltinAndUserFunctionCallsCanCoexist(t *testing.T) {
	errs := checkInput(t, `
		var greet << fct() { return "hi"; };
		show(greet());
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}
