package parser

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
)

func TestParseIndexAssignment(t *testing.T) {
	input := `memory[0x200] << 0xA9u8;`
	p := New(lexer.New(input))
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.IndexAssignStatement)
	if !ok {
		t.Fatalf("statement is not *ast.IndexAssignStatement. got=%T", program.Statements[0])
	}
	if stmt.Target == nil || stmt.Target.Left.String() != "memory" {
		t.Fatalf("unexpected target: %#v", stmt.Target)
	}
	index, ok := stmt.Target.Index.(*ast.IntegerLiteral)
	if !ok || index.Value != 512 {
		t.Fatalf("unexpected index: %#v", stmt.Target.Index)
	}
	value, ok := stmt.Value.(*ast.IntegerLiteral)
	if !ok || value.FixedType != "u8" || value.RawValue != 0xA9 {
		t.Fatalf("unexpected value: %#v", stmt.Value)
	}
}

func TestParseNestedIndexAssignment(t *testing.T) {
	p := New(lexer.New(`matrix[0][1] << 7;`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	stmt, ok := program.Statements[0].(*ast.IndexAssignStatement)
	if !ok {
		t.Fatalf("statement is not index assignment. got=%T", program.Statements[0])
	}
	if _, ok := stmt.Target.Left.(*ast.IndexExpression); !ok {
		t.Fatalf("nested target left is not an index expression. got=%T", stmt.Target.Left)
	}
}
