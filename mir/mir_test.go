package mir_test

import (
	"strings"
	"testing"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/mir"
	"zumbra/parser"
	"zumbra/types"
)

func lower(t *testing.T, source string, optimize bool) *mir.Module {
	t.Helper()
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	info, errs := types.AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type errors: %v", errs)
	}
	h, err := hir.Lower("test.zum", program, info)
	if err != nil {
		t.Fatal(err)
	}
	m, err := mir.Lower(h)
	if err != nil {
		t.Fatal(err)
	}
	if optimize {
		if err := mir.Optimize(m); err != nil {
			t.Fatal(err)
		}
	}
	return m
}

func TestOptimizeFoldsConstantsAndRemovesUnreachableCode(t *testing.T) {
	module := lower(t, `
var calculate << fct() {
    return 2 + 3 * 4;
    show(999);
}
var value << 8 band 3;
`, true)
	dump := module.Dump()
	if !strings.Contains(dump, `const value="14" : int`) {
		t.Fatalf("expected folded 14:\n%s", dump)
	}
	if !strings.Contains(dump, `const value="0" : int`) {
		t.Fatalf("expected folded bitwise result:\n%s", dump)
	}
	if strings.Contains(dump, `value="999"`) {
		t.Fatalf("unreachable instruction remained:\n%s", dump)
	}
}

func TestVerifyRejectsUndefinedValues(t *testing.T) {
	module := &mir.Module{Entry: &mir.Region{Name: "entry", Instructions: []*mir.Instruction{{ID: 1, Op: mir.OpDrop, Args: []mir.ValueID{99}}}}}
	if err := mir.Verify(module); err == nil {
		t.Fatal("expected undefined value error")
	}
}
