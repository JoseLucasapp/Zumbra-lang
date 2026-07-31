package mir

import (
	"testing"
	"zumbra/hir"
	"zumbra/lexer"
	"zumbra/parser"
	"zumbra/types"
)

func TestExternAndUnsafeLowerToMIR(t *testing.T) {
	source := `extern "C" from "native.c" { fct add(a: i32, b: i32) -> i32; } unsafe { add(1i32, 2i32); }`
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	analysis, errs := types.AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("front-end diagnostics: %v %v", p.Errors(), errs)
	}
	h, err := hir.Lower("ffi.zum", program, analysis)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Lower(h)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Declarations) != 1 || module.Declarations[0].Op != OpExtern {
		t.Fatalf("extern declaration missing: %s", module.Dump())
	}
	foundUnsafe := false
	for _, instruction := range module.Entry.Instructions {
		if instruction.Op == OpUnsafe {
			foundUnsafe = true
			if len(instruction.Regions) != 1 {
				t.Fatalf("unsafe region missing: %#v", instruction)
			}
		}
	}
	if !foundUnsafe {
		t.Fatalf("unsafe instruction missing: %s", module.Dump())
	}
}
