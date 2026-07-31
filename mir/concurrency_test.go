package mir

import (
	"strings"
	"testing"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestConcurrencyReachesMIR(t *testing.T) {
	p := parser.New(lexer.New(`fct answer(){42;} var task<<spawn answer(); var result<<await task;`))
	program := p.ParseProgram()
	analysis, errs := types.AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	high, _ := hir.Lower("tasks.zum", program, analysis)
	module, err := Lower(high)
	if err != nil {
		t.Fatal(err)
	}
	dump := module.Dump()
	for _, expected := range []string{" = spawn ", " = await "} {
		if !strings.Contains(dump, expected) {
			t.Fatalf("missing %q:\n%s", expected, dump)
		}
	}
}
