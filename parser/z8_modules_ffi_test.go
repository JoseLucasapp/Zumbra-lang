package parser

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
)

func TestZ8ModuleAndFFISyntax(t *testing.T) {
	input := `
        import "math.zum" as math;
        pub fct add(a, b) { a + b; }
        pub const answer << 42;
        extern "C" from "native.c" {
            fct apply(value: i32, cb: callback(i32) -> i32) -> i32;
        }
        unsafe { show(apply(1i32, fct(x) { x; })); }
    `
	p := New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if len(program.Statements) != 5 {
		t.Fatalf("expected 5 statements, got %d", len(program.Statements))
	}
	imported := program.Statements[0].(*ast.ImportStatement)
	if imported.Alias == nil || imported.Alias.Value != "math" {
		t.Fatalf("missing alias: %#v", imported)
	}
	function := program.Statements[1].(*ast.VarStatement)
	if !function.Public {
		t.Fatal("pub function was not marked public")
	}
	external := program.Statements[3].(*ast.ExternBlockStatement)
	if external.Link != "native.c" || len(external.Functions) != 1 {
		t.Fatalf("bad extern block: %#v", external)
	}
	callback := external.Functions[0].Parameters[1].Type
	if callback.Name != "callback" || callback.CallbackReturn == nil || callback.CallbackReturn.Name != "i32" {
		t.Fatalf("bad callback type: %#v", callback)
	}
	if _, ok := program.Statements[4].(*ast.UnsafeStatement); !ok {
		t.Fatalf("expected unsafe statement, got %T", program.Statements[4])
	}
}
