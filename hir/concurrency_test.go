package hir

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestConcurrencyTypesReachHIR(t *testing.T) {
	p := parser.New(lexer.New(`fct answer(){42;} var task<<spawn answer(); var result<<await task;`))
	program := p.ParseProgram()
	analysis, errs := types.AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), errs)
	}
	module, err := Lower("tasks.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	dump := module.Dump()
	for _, expected := range []string{`spawn : task<int>`, `await : int`} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("missing %q:\n%s", expected, dump)
		}
	}
}
