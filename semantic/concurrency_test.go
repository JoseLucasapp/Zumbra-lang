package semantic

import (
	"testing"

	"zumbra/lexer"
	"zumbra/parser"
)

func TestResolverAcceptsSpawnAndAwait(t *testing.T) {
	p := parser.New(lexer.New(`
        fct answer() { 42; }
        var task << spawn answer();
        await task;
    `))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	_, errs := Analyze(program)
	if len(errs) != 0 {
		t.Fatalf("semantic errors: %v", errs)
	}
}
