package compiler

import (
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
)

func TestCompilerWarningsUnusedVariable(t *testing.T) {
	source := `
	var x << 10;
	`

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	symbolTable := NewSymbolTable()
	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	comp := NewWithStateAndDir(symbolTable, []object.Object{}, ".")
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compile error: %s", err)
	}

	warnings := comp.Warnings()
	if len(warnings) == 0 {
		t.Fatalf("expected warnings, got none")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "variable declared but never used: x") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected unused variable warning for x, got %+v", warnings)
	}
}

func TestCompilerWarningsUnusedImport(t *testing.T) {
	source := `
	import "mod.zum";
	`

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	symbolTable := NewSymbolTable()
	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	comp := NewWithStateAndDir(symbolTable, []object.Object{}, ".")
	_ = comp.Compile(program)

	warnings := comp.Warnings()

	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "import may be unused: mod.zum") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected unused import warning, got %+v", warnings)
	}
}

func TestCompilerWarningsUnreachableCodeAfterReturn(t *testing.T) {
	source := `
	var f << fct() {
		return 10;
		20;
	};
	`

	l := lexer.New(source)
	p := parser.New(l)
	program := p.ParseProgram()

	symbolTable := NewSymbolTable()
	for i, v := range builtins.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	comp := NewWithStateAndDir(symbolTable, []object.Object{}, ".")
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compile error: %s", err)
	}

	warnings := comp.Warnings()

	found := false
	for _, w := range warnings {
		if strings.Contains(w.Message, "unreachable code detected") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected unreachable code warning, got %+v", warnings)
	}
}
