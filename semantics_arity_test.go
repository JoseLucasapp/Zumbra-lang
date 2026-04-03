package main

import (
	"strings"
	"testing"

	"zumbra/compiler"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/parser"
)

func TestArityValidation_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantCompileErr string
	}{
		{
			name: "named function wrong arity",
			source: `
			var add << fct(a, b) {
				a + b;
			};

			add(1);
			`,
			wantCompileErr: "wrong number of arguments for add: want=2, got=1",
		},
		{
			name: "async function wrong arity",
			source: `
			var task << async fct(a, b) {
				a + b;
			};

			await task(1);
			`,
			wantCompileErr: "wrong number of arguments for task: want=2, got=1",
		},
		{
			name: "anonymous function wrong arity",
			source: `
			(fct(a, b) {
				a + b;
			})(1);
			`,
			wantCompileErr: "wrong number of arguments for anonymous function: want=2, got=1",
		},
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
			if err == nil {
				t.Fatalf("expected compile error %q, got nil", tt.wantCompileErr)
			}

			if err.Error() != tt.wantCompileErr {
				t.Fatalf("wrong compile error. want=%q, got=%q", tt.wantCompileErr, err.Error())
			}
		})
	}
}
