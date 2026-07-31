package types

import (
	"strings"
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func parseContextualCallbackProgram(t *testing.T, source string) *ast.Program {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return program
}

func callbackLiteralFromVar(t *testing.T, program *ast.Program, name string) *ast.FunctionLiteral {
	t.Helper()
	for _, statement := range program.Statements {
		variable, ok := statement.(*ast.VarStatement)
		if !ok || variable.Name == nil || variable.Name.Value != name {
			continue
		}
		function, ok := variable.Value.(*ast.FunctionLiteral)
		if !ok {
			t.Fatalf("%s is not a function literal: %T", name, variable.Value)
		}
		return function
	}
	t.Fatalf("function variable %s not found", name)
	return nil
}

func TestContextualCallbackRefinesNamedFunction(t *testing.T) {
	program := parseContextualCallbackProgram(t, `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var double << fct(value) { value * 2i32; };
        unsafe { apply(21i32, double); }
    `)
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(diagnostics) != 0 {
		t.Fatalf("type diagnostics: %v", diagnostics)
	}

	doubleType, ok := analysis.Global("double")
	if !ok || doubleType.String() != "fct(i32) -> i32" {
		t.Fatalf("unexpected callback type: %#v", doubleType)
	}
	literal := callbackLiteralFromVar(t, program, "double")
	if got := analysis.TypeOf(literal).String(); got != "fct(i32) -> i32" {
		t.Fatalf("function literal was not refined: %s", got)
	}
}

func TestContextualCallbackRefinesInlineFunction(t *testing.T) {
	program := parseContextualCallbackProgram(t, `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        unsafe { apply(21i32, fct(value) { value * 2i32; }); }
    `)
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(diagnostics) != 0 {
		t.Fatalf("type diagnostics: %v", diagnostics)
	}

	unsafeStatement := program.Statements[1].(*ast.UnsafeStatement)
	expression := unsafeStatement.Body.Statements[0].(*ast.ExpressionStatement)
	call := expression.Expression.(*ast.CallExpression)
	literal := call.Arguments[1].(*ast.FunctionLiteral)
	if got := analysis.TypeOf(literal).String(); got != "fct(i32) -> i32" {
		t.Fatalf("inline callback was not refined: %s", got)
	}
}

func TestContextualCallbackRejectsReturnMismatch(t *testing.T) {
	program := parseContextualCallbackProgram(t, `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var wrong << fct(value) { "not-an-int"; };
        unsafe { apply(21i32, wrong); }
    `)
	_, diagnostics := AnalyzeWithInfo(program)
	if len(diagnostics) == 0 {
		t.Fatal("expected callback return diagnostic")
	}
	joined := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		joined = append(joined, diagnostic.Error())
	}
	if !strings.Contains(strings.Join(joined, "\n"), "callback return expects i32, got string") {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestContextualCallbackRejectsArityMismatch(t *testing.T) {
	program := parseContextualCallbackProgram(t, `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var wrong << fct(left, right) { left + right; };
        unsafe { apply(21i32, wrong); }
    `)
	_, diagnostics := AnalyzeWithInfo(program)
	if len(diagnostics) == 0 {
		t.Fatal("expected callback arity diagnostic")
	}
	joined := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		joined = append(joined, diagnostic.Error())
	}
	if !strings.Contains(strings.Join(joined, "\n"), "callback expects 1 parameters, got 2") {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}

func TestContextualCallbackCannotBeReusedWithIncompatibleSignatures(t *testing.T) {
	program := parseContextualCallbackProgram(t, `
        extern "C" {
            fct applyInt(value: i32, cb: callback(i32) -> i32) -> i32;
            fct applyText(value: cstring, cb: callback(cstring) -> cstring) -> cstring;
        }
        var identity << fct(value) { value; };
        unsafe {
            applyInt(21i32, identity);
            applyText("zumbra", identity);
        }
    `)
	_, diagnostics := AnalyzeWithInfo(program)
	if len(diagnostics) == 0 {
		t.Fatal("expected incompatible contextual signature diagnostic")
	}
	joined := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		joined = append(joined, diagnostic.Error())
	}
	if !strings.Contains(strings.Join(joined, "\n"), "expects fct(string) -> string, got fct(i32) -> i32") {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}
