package semantic

import (
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestResolveIndexAssignment(t *testing.T) {
	p := parser.New(lexer.New(`
		var memory << [0, 0];
		memory[1] << 42;
	`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	resolver := NewResolver()
	if errs := resolver.Resolve(program); len(errs) != 0 {
		t.Fatalf("resolver errors: %v", errs)
	}
}
