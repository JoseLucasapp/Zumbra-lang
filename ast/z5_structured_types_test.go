package ast_test

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ5ASTString(t *testing.T) {
	p := parser.New(lexer.New(`const Max << 3; enum Direction { Up; Down; }`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}
	got := program.String()
	if !strings.Contains(got, "const Max << 3") || !strings.Contains(got, "enum Direction") {
		t.Fatalf("unexpected AST string: %s", got)
	}
}
