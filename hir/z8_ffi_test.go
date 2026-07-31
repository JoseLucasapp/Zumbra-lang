package hir

import (
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestExternAndUnsafeLowerToHIR(t *testing.T) {
	source := `extern "C" from "native.c" { fct add(a: i32, b: i32) -> i32; } unsafe { add(1i32, 2i32); }`
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	analysis, errs := types.AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type diagnostics: %v", errs)
	}
	module, err := Lower("ffi.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Root.Children) != 2 || module.Root.Children[0].Kind != ExternKind || module.Root.Children[1].Kind != UnsafeKind {
		t.Fatalf("unexpected HIR: %s", module.Dump())
	}
	external := module.Root.Children[0].Children[0]
	if external.Kind != ExternFuncKind || external.Meta["link"] != "native.c" || external.Meta["param.0.type"] != "i32" {
		t.Fatalf("extern metadata missing: %#v", external.Meta)
	}
}
