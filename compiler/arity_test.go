package compiler

import (
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
)

func compileSourceForArity(t *testing.T, source string) error {
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
	return comp.Compile(program)
}

func TestCompilerArityValidationNamedFunction(t *testing.T) {
	err := compileSourceForArity(t, `
	var add << fct(a, b) {
		a + b;
	};

	add(1);
	`)

	if err == nil {
		t.Fatalf("expected arity error, got nil")
	}

	expected := "wrong number of arguments for add: want=2, got=1"
	if err.Error() != expected {
		t.Fatalf("wrong error. want=%q, got=%q", expected, err.Error())
	}
}

func TestCompilerArityValidationAsyncFunction(t *testing.T) {
	err := compileSourceForArity(t, `
	var task << async fct(a, b) {
		a + b;
	};

	await task(1);
	`)

	if err == nil {
		t.Fatalf("expected arity error, got nil")
	}

	expected := "wrong number of arguments for task: want=2, got=1"
	if err.Error() != expected {
		t.Fatalf("wrong error. want=%q, got=%q", expected, err.Error())
	}
}

func TestCompilerArityValidationAnonymousFunctionCall(t *testing.T) {
	err := compileSourceForArity(t, `
	(fct(a, b) {
		a + b;
	})(1);
	`)

	if err == nil {
		t.Fatalf("expected arity error, got nil")
	}

	expected := "wrong number of arguments for anonymous function: want=2, got=1"
	if err.Error() != expected {
		t.Fatalf("wrong error. want=%q, got=%q", expected, err.Error())
	}
}

func TestCompilerArityValidationCorrectCallPasses(t *testing.T) {
	err := compileSourceForArity(t, `
	var add << fct(a, b) {
		a + b;
	};

	add(1, 2);
	`)

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}
