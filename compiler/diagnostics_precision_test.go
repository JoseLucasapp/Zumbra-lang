package compiler

import (
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
)

func compileForWarnings(t *testing.T, source string) []Diagnostic {
	t.Helper()

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors:\n\t%s", strings.Join(p.Errors(), "\n\t"))
	}

	symbolTable := NewSymbolTable()
	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	comp := NewWithStateAndDir(symbolTable, []object.Object{}, ".")
	if err := comp.Compile(program); err != nil {
		t.Fatalf("compile error: %s", err)
	}

	return comp.Warnings()
}

func TestCompilerWarnsUnusedParameterSeparately(t *testing.T) {
	warnings := compileForWarnings(t, `
	var f << fct(a, b) {
		a;
	};
	`)

	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "parameter declared but never used: b") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected unused parameter warning, got %+v", warnings)
	}
}

func TestCompilerDoesNotWarnSyntheticVariables(t *testing.T) {
	warnings := compileForWarnings(t, `
	var total << 0;
	for i in 1..3 {
		total << total + i;
	}
	total;
	`)

	for _, w := range warnings {
		if strings.Contains(w.Message, "__z_") {
			t.Fatalf("unexpected synthetic warning: %+v", warnings)
		}
	}
}
