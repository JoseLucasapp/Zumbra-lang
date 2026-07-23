package mir

import (
	"strings"
	"testing"

	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestOrdinaryCallsRefineMIRFunctionTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var identity << fct(value) { value; };
        var result << identity(42);
    `))
	program := p.ParseProgram()
	analysis, diagnostics := types.AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	high, err := hir.Lower("call_inference.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Lower(high)
	if err != nil {
		t.Fatal(err)
	}
	dump := module.Dump()
	for _, expected := range []string{
		`function identity(value) -> int`,
		`function_ref "identity" : fct(int) -> int`,
		`call`,
		`: int`,
	} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("missing %q:\n%s", expected, dump)
		}
	}
	if strings.Contains(dump, `fct(unknown) -> unknown`) {
		t.Fatalf("MIR retained unknown function signature:\n%s", dump)
	}
}
