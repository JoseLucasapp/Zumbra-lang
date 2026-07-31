package main

import (
	"strings"
	"testing"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
	"zumbra/vm"
)

func TestOperatorAndConditionalSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
	}{
		{"addition", `1 + 2;`, "3", "", ""},
		{"subtraction", `10 - 3;`, "7", "", ""},
		{"multiplication", `2 * 3;`, "6", "", ""},
		{"division", `8 / 2;`, "4", "", ""},
		{"modulo", `7 % 4;`, "3", "", ""},
		{"power", `2 ** 3;`, "8", "", ""},

		{"less than", `1 < 2;`, "true", "", ""},
		{"less or equal", `1 <= 1;`, "true", "", ""},
		{"greater than", `2 > 1;`, "true", "", ""},
		{"greater or equal", `2 >= 2;`, "true", "", ""},
		{"equals", `1 == 1;`, "true", "", ""},
		{"not equals", `1 != 2;`, "true", "", ""},

		{"boolean and", `true and false;`, "false", "", ""},
		{"boolean or", `true or false;`, "true", "", ""},
		{"boolean not", `!false;`, "true", "", ""},

		{"precedence multiply before add", `1 + 2 * 3;`, "7", "", ""},
		{"precedence grouped expression", `(1 + 2) * 3;`, "9", "", ""},
		{"precedence right associative power", `2 ** 3 ** 2;`, "512", "", ""},

		{"if true branch", `if (true) { 10; } else { 20; }`, "10", "", ""},
		{"if false branch", `if (false) { 10; } else { 20; }`, "20", "", ""},
		{"if comparison", `if (1 < 2) { 10; } else { 20; }`, "10", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.source)
			p := parser.New(l)
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("parser errors:\n\t%s", strings.Join(p.Errors(), "\n\t"))
			}

			symbolTable := compiler.NewSymbolTable()
			for i, v := range builtins.Builtins {
				symbolTable.DefineBuiltin(i, v.Name)
			}

			comp := compiler.NewWithStateAndDir(symbolTable, []object.Object{}, ".")
			err := comp.Compile(program)

			if tt.wantCompileErr != "" {
				if err == nil {
					t.Fatalf("expected compile error %q, got nil", tt.wantCompileErr)
				}
				if err.Error() != tt.wantCompileErr {
					t.Fatalf("wrong compile error. want=%q, got=%q", tt.wantCompileErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected compile error: %s", err)
			}

			machine := vm.New(comp.Bytecode())
			err = machine.Run()

			if tt.wantRunErr != "" {
				if err == nil {
					t.Fatalf("expected run error %q, got nil", tt.wantRunErr)
				}
				if err.Error() != tt.wantRunErr {
					t.Fatalf("wrong run error. want=%q, got=%q", tt.wantRunErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected run error: %s", err)
			}

			got := machine.LastPoppedStackElem()
			if got == nil {
				t.Fatalf("last popped stack elem is nil")
			}

			if got.Inspect() != tt.wantResult {
				t.Fatalf("wrong result. want=%q, got=%q", tt.wantResult, got.Inspect())
			}
		})
	}
}
