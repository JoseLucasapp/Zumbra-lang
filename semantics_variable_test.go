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

func TestVariableAndScopeSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
	}{
		{
			name: "global declaration and read",
			source: `
			var x << 10;
			x;
			`,
			wantResult: "10",
		},
		{
			name: "global reassignment",
			source: `
			var x << 10;
			x << 30;
			x;
			`,
			wantResult: "30",
		},
		{
			name: "function local shadowing",
			source: `
			var x << 10;
			var f << fct() {
				var x << 20;
				x;
			};
			f();
			`,
			wantResult: "20",
		},
		{
			name: "outer global preserved after shadowing",
			source: `
			var x << 10;
			var f << fct() {
				var x << 20;
				x;
			};
			f();
			x;
			`,
			wantResult: "10",
		},
		{
			name: "read outer variable from function",
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
			name: "reassign outer variable from function",
			source: `
			var x << 10;
			var f << fct() {
				x << 20;
			};
			f();
			x;
			`,
			wantResult: "20",
		},
		{
			name: "undefined variable compile error",
			source: `
			missing;
			`,
			wantCompileErr: "undefined variable missing",
		},
		{
			name: "if block local variable is not visible outside",
			source: `
			if (true) {
				var x << 10;
			}
			x;
			`,
			wantCompileErr: "undefined variable x",
		},
		{
			name: "if block shadowing preserves outer variable",
			source: `
			var x << 10;
			if (true) {
				var x << 20;
				x;
			}
			x;
			`,
			wantResult: "10",
		},
		{
			name: "if block can reassign outer variable",
			source: `
			var x << 10;
			if (true) {
				x << 20;
			}
			x;
			`,
			wantResult: "20",
		},
		{
			name: "while block local variable is not visible outside",
			source: `
			var done << false;
			while (done == false) {
				var x << 10;
				done << true;
			}
			x;
			`,
			wantCompileErr: "undefined variable x",
		},
		{
			name: "while block can reassign outer variable",
			source: `
			var x << 10;
			var done << false;
			while (done == false) {
				x << 20;
				done << true;
			}
			x;
			`,
			wantResult: "20",
		},
		{
			name: "for range block local variable is not visible outside",
			source: `
			for i in 1..2 {
				var x << i;
			}
			x;
			`,
			wantCompileErr: "undefined variable x",
		},
		{
			name: "for range block can reassign outer variable",
			source: `
			var x << 0;
			for i in 1..3 {
				x << i;
			}
			x;
			`,
			wantResult: "3",
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
