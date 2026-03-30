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

func TestCoreSyntaxSnippetsParseAndCompile(t *testing.T) {
	snippets := []string{
		`var x << 10; x << x + 1;`,
		`var sum << fct(a, b) { a + b; }; sum(1, 2);`,
		`if (true) { 1; } else { 2; }`,
		`while (false) { 1; }`,
		`for i in 1..3 { i; }`,
		`var arr << [1, 2, 3]; arr[0];`,
		`var dict << {"name": "z"}; dict["name"];`,
		`var task << async fct() { 10; }; await task();`,
		`var run << fct() { 1; }; try run() or err { err; };`,
	}

	for i, source := range snippets {
		t.Run(strings.Join([]string{"snippet", string(rune('0' + i))}, "_"), func(t *testing.T) {
			l := lexer.New(source)
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
			if err := comp.Compile(program); err != nil {
				t.Fatalf("compiler error: %s", err)
			}
		})
	}
}
