package parser

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
)

func TestParseZ5DeclarationsAndMatch(t *testing.T) {
	input := `
        const Max << 3;
        type Byte << u8;
        struct Point {
            x: int;
            y: int;
            fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; }
        }
        enum Direction { Up; Down; }
        var p << Point(1, 2);
        p.x << 4;
        match(Direction.Up) { case Direction.Up { 1; } else { 0; } };
    `
	p := New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if len(program.Statements) != 7 {
		t.Fatalf("expected 7 statements, got %d", len(program.Statements))
	}
	st, ok := program.Statements[2].(*ast.StructStatement)
	if !ok {
		t.Fatalf("expected struct, got %T", program.Statements[2])
	}
	if len(st.Fields) != 2 || len(st.Methods) != 1 {
		t.Fatalf("unexpected struct: %#v", st)
	}
	if len(st.Methods[0].Function.Parameters) != 3 || st.Methods[0].Function.Parameters[0].Value != "self" {
		t.Fatalf("self was not inserted")
	}
	if _, ok := program.Statements[5].(*ast.AttributeAssignStatement); !ok {
		t.Fatalf("expected field assignment, got %T", program.Statements[5])
	}
	expr := program.Statements[6].(*ast.ExpressionStatement).Expression
	if _, ok := expr.(*ast.MatchExpression); !ok {
		t.Fatalf("expected match, got %T", expr)
	}
}
