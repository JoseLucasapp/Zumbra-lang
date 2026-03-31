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

func TestFunctionAndClosureSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
	}{
		{
			name: "implicit return uses last expression",
			source: `
			var f << fct() {
				10;
			};
			f();
			`,
			wantResult: "10",
		},
		{
			name: "explicit return works",
			source: `
			var f << fct() {
				return 20;
			};
			f();
			`,
			wantResult: "20",
		},
		{
			name: "last expression wins",
			source: `
			var f << fct() {
				10;
				30;
			};
			f();
			`,
			wantResult: "30",
		},
		{
			name: "closure captures outer variable",
			source: `
			var x << 10;
			var f << fct() {
				x;
			};
			f();
			`,
			wantResult: "10",
		},
		{
			name: "closure factory works",
			source: `
			var makeAdder << fct(a) {
				fct(b) {
					a + b;
				};
			};

			var addTwo << makeAdder(2);
			addTwo(5);
			`,
			wantResult: "7",
		},
		{
			name: "nested closure works",
			source: `
			var outer << fct(a) {
				fct(b) {
					fct(c) {
						a + b + c;
					};
				};
			};

			var level1 << outer(1);
			var level2 << level1(2);
			level2(3);
			`,
			wantResult: "6",
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
