package semantic

import (
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestUserDeclarationMayShadowBuiltin(t *testing.T) {
	p := parser.New(lexer.New(`var sum << fct(a, b) { a + b; }; sum(1, 2);`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	_, errs := Analyze(program)
	if len(errs) != 0 {
		t.Fatalf("semantic errors: %v", errs)
	}
}
