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

func TestErrorSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
	}{
		{
			name: "try passes through non error value",
			source: `
			try 10;
			`,
			wantResult: "10",
		},
		{
			name: "error handler block handles error",
			source: `
			try error("boom") or {
				20;
			};
			`,
			wantResult: "20",
		},
		{
			name: "error handler with binding exposes error",
			source: `
			try error("boom") or err {
				err;
			};
			`,
			wantResult: "ERROR: boom",
		},
		{
			name: "or block does not run on success",
			source: `
			try 10 or {
				20;
			};
			`,
			wantResult: "10",
		},
		{
			name: "function can return error object",
			source: `
			var f << fct() {
				error("boom");
			};
			f();
			`,
			wantResult: "ERROR: boom",
		},
		{
			name: "function local handler recovers error",
			source: `
			var f << fct() {
				try error("boom") or {
					30;
				};
			};
			f();
			`,
			wantResult: "30",
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
