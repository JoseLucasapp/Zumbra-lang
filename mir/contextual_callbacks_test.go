package mir

import (
	"strings"
	"testing"

	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestContextualCallbackTypesReachMIR(t *testing.T) {
	source := `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var double << fct(value) { value * 2i32; };
        unsafe { apply(21i32, double); }
    `
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	analysis, diagnostics := types.AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("front-end diagnostics: %v %v", p.Errors(), diagnostics)
	}
	h, err := hir.Lower("callbacks.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Lower(h)
	if err != nil {
		t.Fatal(err)
	}
	dump := module.Dump()
	for _, expected := range []string{
		`function double(value) -> i32`,
		`load "value" : i32`,
		`binary op="*"`,
		`: i32`,
		`function_ref "double" : fct(i32) -> i32`,
		`load "double" : fct(i32) -> i32`,
	} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("MIR does not contain %q:\n%s", expected, dump)
		}
	}
}
