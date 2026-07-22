package mir

import (
	"testing"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestZ7MIRPreservesMethodOwnerAndParameters(t *testing.T) {
	p := parser.New(lexer.New(`struct Counter { value: int; fct add(amount) { self.value << self.value + amount; } }`))
	program := p.ParseProgram()
	analysis, errs := types.AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type errors: %v", errs)
	}
	h, err := hir.Lower("test.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Lower(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Functions) != 1 {
		t.Fatalf("expected one method, got %d", len(module.Functions))
	}
	method := module.Functions[0]
	if method.Owner != "Counter" {
		t.Fatalf("unexpected owner: %q", method.Owner)
	}
	if got := method.Parameters; len(got) != 2 || got[0] != "self" || got[1] != "amount" {
		t.Fatalf("unexpected parameters: %v", got)
	}
}
