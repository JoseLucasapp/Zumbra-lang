package hir

import (
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestZ7HIRPreservesFunctionParametersAndMethodOwner(t *testing.T) {
	input := `struct Counter { value: int; fct add(amount) { self.value << self.value + amount; } }`
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	analysis, errs := types.AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type errors: %v", errs)
	}
	module, err := Lower("test.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	method := module.Root.Children[0].Children[1]
	if method.Meta["owner"] != "Counter" {
		t.Fatalf("owner not preserved: %#v", method.Meta)
	}
	if method.Meta["params"] != "self,amount" {
		t.Fatalf("parameters not preserved: %#v", method.Meta)
	}
}
