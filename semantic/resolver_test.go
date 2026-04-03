package semantic

import (
	"fmt"
	"strings"
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func parse(input string) *parser.Parser {
	l := lexer.New(input)
	return parser.New(l)
}

func parseProgramOrFatal(t *testing.T, input string) *ast.Program {
	t.Helper()

	p := parse(input)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	return program
}

func resolveInput(input string) []error {
	p := parse(input)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return []error{fmt.Errorf("%s", p.Errors()[0])}
	}

	r := NewResolver()
	return r.Resolve(program)
}

func findFunctionLiteralInVar(t *testing.T, program *ast.Program, varName string) *ast.FunctionLiteral {
	t.Helper()

	for _, stmt := range program.Statements {
		varStmt, ok := stmt.(*ast.VarStatement)
		if !ok || varStmt.Name == nil || varStmt.Name.Value != varName {
			continue
		}

		fn, ok := varStmt.Value.(*ast.FunctionLiteral)
		if !ok {
			t.Fatalf("variable %q is not a function literal", varName)
		}

		return fn
	}

	t.Fatalf("function variable %q not found", varName)
	return nil
}

func freeSymbolNames(fr FunctionResolution) []string {
	names := make([]string, 0, len(fr.FreeSymbols))
	for _, sym := range fr.FreeSymbols {
		names = append(names, sym.Name)
	}
	return names
}

func containsName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func TestResolveUndefinedIdentifier(t *testing.T) {
	errs := resolveInput(`foobar;`)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "undefined symbol: foobar") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestResolveVarDeclarationAndUsage(t *testing.T) {
	errs := resolveInput(`
		var x << 10;
		x;
	`)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveDuplicateVarInSameScope(t *testing.T) {
	errs := resolveInput(`
		var x << 1;
		var x << 2;
	`)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "symbol already declared in this scope: x") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestResolveNestedScopeLookup(t *testing.T) {
	errs := resolveInput(`
		var x << 10;
		if (true) { x; }
	`)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveBuiltinUsage(t *testing.T) {
	errs := resolveInput(`
		show("hello");
		sizeOf([1, 2, 3]);
	`)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveDuplicateFunctionParameters(t *testing.T) {
	errs := resolveInput(`
		var add << fct(a, a) { return a; };
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "symbol already declared in this scope: a") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestResolveFunctionCanAccessOuterScope(t *testing.T) {
	errs := resolveInput(`
		var x << 10;
		var add << fct(a) { return a + x; };
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveShadowingInInnerScope(t *testing.T) {
	errs := resolveInput(`
		var x << 10;
		if (true) {
			var x << 20;
			x;
		}
		x;
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveNamedFunctionDeclarationViaVar(t *testing.T) {
	errs := resolveInput(`
		var add << fct(a, b) { return a + b; };
		add(1, 2);
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveUndefinedIdentifierInsideFunction(t *testing.T) {
	errs := resolveInput(`
		var add << fct(a) { return a + x; };
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	if !strings.Contains(errs[0].Error(), "undefined symbol: x") {
		t.Fatalf("unexpected error: %v", errs[0])
	}
}

func TestResolveBuiltinInsideFunction(t *testing.T) {
	errs := resolveInput(`
		var demo << fct() {
			show("ok");
		};
	`)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

func TestResolveFunctionFreeSymbols(t *testing.T) {
	input := `
		var x << 10;
		var add << fct(a) { return a + x; };
	`

	program := parseProgramOrFatal(t, input)
	r := NewResolver()
	errs := r.Resolve(program)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}

	fn := findFunctionLiteralInVar(t, program, "add")
	resolution, ok := r.Result().Functions[fn]
	if !ok {
		t.Fatalf("expected function resolution for add")
	}

	names := freeSymbolNames(resolution)
	if !containsName(names, "x") {
		t.Fatalf("expected free symbol x, got %v", names)
	}
	if containsName(names, "a") {
		t.Fatalf("parameter a must not be marked as free, got %v", names)
	}
}

func TestResolveNestedFunctionFreeSymbols(t *testing.T) {
	input := `
		var x << 1;
		var outer << fct() {
			var y << 2;
			var inner << fct() {
				return x + y;
			};
		};
	`

	program := parseProgramOrFatal(t, input)
	r := NewResolver()
	errs := r.Resolve(program)

	if len(errs) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errs), errs)
	}

	outerFn := findFunctionLiteralInVar(t, program, "outer")
	outerRes, ok := r.Result().Functions[outerFn]
	if !ok {
		t.Fatalf("expected function resolution for outer")
	}

	outerNames := freeSymbolNames(outerRes)
	if len(outerNames) != 0 {
		t.Fatalf("outer should not have free symbols, got %v", outerNames)
	}

	var innerFn *ast.FunctionLiteral
	for _, stmt := range outerFn.Body.Statements {
		varStmt, ok := stmt.(*ast.VarStatement)
		if !ok || varStmt.Name == nil || varStmt.Name.Value != "inner" {
			continue
		}

		fn, ok := varStmt.Value.(*ast.FunctionLiteral)
		if !ok {
			t.Fatalf("inner is not a function literal")
		}
		innerFn = fn
		break
	}

	if innerFn == nil {
		t.Fatalf("inner function not found")
	}

	innerRes, ok := r.Result().Functions[innerFn]
	if !ok {
		t.Fatalf("expected function resolution for inner")
	}

	innerNames := freeSymbolNames(innerRes)
	if !containsName(innerNames, "x") {
		t.Fatalf("expected inner to capture x, got %v", innerNames)
	}
	if !containsName(innerNames, "y") {
		t.Fatalf("expected inner to capture y, got %v", innerNames)
	}
}

func TestResolveUndefinedIdentifierIncludesPosition(t *testing.T) {
	errs := resolveInput(`
		foobar;
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	msg := errs[0].Error()
	if !strings.Contains(msg, "undefined symbol: foobar") {
		t.Fatalf("unexpected message: %s", msg)
	}
	if !strings.Contains(msg, "line") || !strings.Contains(msg, "col") {
		t.Fatalf("expected line/col in message, got: %s", msg)
	}
}

func TestResolveDuplicateParameterIncludesPosition(t *testing.T) {
	errs := resolveInput(`
		var add << fct(a, a) { return a; };
	`)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	msg := errs[0].Error()
	if !strings.Contains(msg, "symbol already declared in this scope: a") {
		t.Fatalf("unexpected message: %s", msg)
	}
	if !strings.Contains(msg, "line") || !strings.Contains(msg, "col") {
		t.Fatalf("expected line/col in message, got: %s", msg)
	}
}
