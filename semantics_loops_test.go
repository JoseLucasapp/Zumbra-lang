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

func TestLoopSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
	}{
		{
			name: "break exits range loop",
			source: `
			var x << 0;
			for i in 1..10 {
				if (i == 4) {
					break;
				}
				x << i;
			}
			x;
			`,
			wantResult: "3",
		},
		{
			name: "continue skips current iteration",
			source: `
			var x << 0;
			for i in 1..5 {
				if (i == 3) {
					continue;
				}
				x << i;
			}
			x;
			`,
			wantResult: "5",
		},
		{
			name: "forever loop can break",
			source: `
			var done << false;
			var x << 0;
			for {
				x << x + 1;
				done << true;
				if (done == true) {
					break;
				}
			}
			x;
			`,
			wantResult: "1",
		},
		{
			name: "foreach array loop sums values",
			source: `
			var total << 0;
			var arr << [1, 2, 3];
			for item in arr {
				total << total + item;
			}
			total;
			`,
			wantResult: "6",
		},
		{
			name: "foreach dict loop sums values",
			source: `
			var total << 0;
			var dict << {"a": 1, "b": 2};
			for key, value in dict {
				total << total + value;
			}
			total;
			`,
			wantResult: "3",
		},
		{
			name: "range where filter works",
			source: `
			var total << 0;
			for i in 1..5 where i > 2 {
				total << total + i;
			}
			total;
			`,
			wantResult: "12",
		},
		{
			name: "array where filter works",
			source: `
			var total << 0;
			var arr << [1, 2, 3, 4];
			for item in arr where item > 2 {
				total << total + item;
			}
			total;
			`,
			wantResult: "7",
		},
		{
			name: "dict where filter works",
			source: `
			var total << 0;
			var dict << {"a": 1, "b": 2, "c": 3};
			for key, value in dict where value > 1 {
				total << total + value;
			}
			total;
			`,
			wantResult: "5",
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
