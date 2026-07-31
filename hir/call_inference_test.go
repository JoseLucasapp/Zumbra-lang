package hir

import (
	"strings"
	"testing"

	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestOrdinaryCallsRefineHIRFunctionTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var identity << fct(value) { value; };
        var result << identity(42);
    `))
	program := p.ParseProgram()
	analysis, diagnostics := types.AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	module, err := Lower("call_inference.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	dump := module.Dump()
	for _, expected := range []string{
		`var name="identity" : fct(int) -> int`,
		`function name="identity" : fct(int) -> int`,
		`identifier name="value" : int`,
		`call : int`,
	} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("missing %q:\n%s", expected, dump)
		}
	}
}
