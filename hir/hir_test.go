package hir_test

import (
	"strings"
	"testing"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestLowerProducesTypedDeterministicHIR(t *testing.T) {
	source := `
struct Point { x: int; y: int; }
var point << Point(2, 3);
var total << point.x + point.y;
`
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	info, errs := types.AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type errors: %v", errs)
	}
	module, err := hir.Lower("test.zum", program, info)
	if err != nil {
		t.Fatal(err)
	}
	dump := module.Dump()
	for _, wanted := range []string{"struct name=\"Point\" : Point", "binary op=\"+\" : int", "field name=\"x\" : int"} {
		if !strings.Contains(dump, wanted) {
			t.Fatalf("HIR missing %q:\n%s", wanted, dump)
		}
	}
	if strings.Contains(dump, " unknown value=\"*") {
		t.Fatalf("unexpected unknown HIR node:\n%s", dump)
	}
	if dump != module.Dump() {
		t.Fatal("HIR dump must be deterministic")
	}
}
