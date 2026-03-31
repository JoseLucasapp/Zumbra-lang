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

func TestCollectionsAndAccessSemantics_CurrentOfficialPath(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		wantResult     string
		wantCompileErr string
		wantRunErr     string
		wantErrorType  string
	}{
		{
			name: "array index access",
			source: `
			var arr << [1, 2, 3];
			arr[0];
			`,
			wantResult: "1",
		},
		{
			name: "array index expression",
			source: `
			var arr << [1, 2, 3];
			arr[1] + arr[2];
			`,
			wantResult: "5",
		},
		{
			name: "dict string key access",
			source: `
			var user << {"name": "Lucas", "age": 25};
			user["name"];
			`,
			wantResult: "Lucas",
		},
		{
			name: "dict numeric field via string key",
			source: `
			var user << {"name": "Lucas", "age": 25};
			user["age"];
			`,
			wantResult: "25",
		},
		{
			name: "attribute access on dict-like object",
			source: `
			var user << {"name": "Lucas", "age": 25};
			user.name;
			`,
			wantResult: "Lucas",
		},
		{
			name: "nested attribute access",
			source: `
			var user << {"profile": {"name": "Lucas"}};
			user.profile.name;
			`,
			wantResult: "Lucas",
		},
		{
			name: "nested index access",
			source: `
			var user << {"profile": {"name": "Lucas"}};
			user["profile"]["name"];
			`,
			wantResult: "Lucas",
		},
		{
			name: "mixed attribute and index access",
			source: `
			var user << {"profile": {"name": "Lucas"}};
			user.profile["name"];
			`,
			wantResult: "Lucas",
		},
		{
			name: "array of objects access",
			source: `
			var user << {"items": [{"name": "a"}, {"name": "b"}]};
			user.items[1].name;
			`,
			wantResult: "b",
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
