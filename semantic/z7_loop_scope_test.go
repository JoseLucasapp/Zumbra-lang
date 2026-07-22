package semantic

import (
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ7WhereClauseCanUseIterationVariable(t *testing.T) {
	input := `var values << [1, 2, 3]; for item in values where item % 2 == 0 { show(item); }`
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	_, errs := Analyze(program)
	if len(errs) != 0 {
		t.Fatalf("semantic errors: %v", errs)
	}
}
