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

func TestAsyncSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
		wantErrorMsg   string
	}{
		{
			name: "await simple async result",
			source: `
			var task << async fct() {
				10;
			};
			await task();
			`,
			wantResult: "10",
		},
		{
			name: "await explicit return",
			source: `
			var task << async fct() {
				return 20;
			};
			await task();
			`,
			wantResult: "20",
		},
		{
			name: "await async expression result",
			source: `
			var task << async fct() {
				10 + 5;
			};
			await task();
			`,
			wantResult: "15",
		},
		{
			name: "await async with parameters",
			source: `
			var task << async fct(a, b) {
				a + b;
			};
			await task(3, 4);
			`,
			wantResult: "7",
		},
		{
			name: "await participates in outer expression",
			source: `
			var task << async fct() {
				10;
			};
			await task() + 2;
			`,
			wantResult: "12",
		},
		{
			name: "try await recovers async error",
			source: `
			var task << async fct() {
				error("boom");
			};
			try await task() or {
				30;
			};
			`,
			wantResult: "30",
		},
		{
			name: "await async error returns error object",
			source: `
			var task << async fct() {
				error("boom");
			};
			await task();
			`,
			wantErrorMsg: "boom",
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

			if tt.wantErrorMsg != "" {
				errObj, ok := got.(*object.Error)
				if !ok {
					t.Fatalf("object is not Error. got=%T (%+v)", got, got)
				}
				if errObj.Message != tt.wantErrorMsg {
					t.Fatalf("wrong error message. want=%q, got=%q", tt.wantErrorMsg, errObj.Message)
				}
				return
			}

			if got.Inspect() != tt.wantResult {
				t.Fatalf("wrong result. want=%q, got=%q", tt.wantResult, got.Inspect())
			}
		})
	}
}
