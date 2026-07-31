package types

import (
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestExternTypesAndCallbackCompatibility(t *testing.T) {
	source := `
        extern "C" {
            fct apply(value: i32, callback: callback(i32) -> i32) -> i32;
            fct address() -> ptr;
        }
        var double << fct(value) { value * 2i32; };
        unsafe { apply(21i32, double); address(); }
    `
	p := parser.New(lexer.New(source))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	analysis, errs := AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type diagnostics: %v", errs)
	}
	apply, ok := analysis.Global("apply")
	if !ok || apply.Kind != Func || len(apply.Params) != 2 || apply.Params[1].Kind != Func {
		t.Fatalf("unexpected apply type: %#v", apply)
	}
	address, ok := analysis.Global("address")
	if !ok || address.Return == nil || address.Return.Kind != Pointer {
		t.Fatalf("unexpected pointer return: %#v", address)
	}
}
