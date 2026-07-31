package parser

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
)

func TestNestedMixedAccessAstShape(t *testing.T) {
	input := `user.items[1].name;`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program should contain 1 statement, got=%d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("statement is not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	attr, ok := stmt.Expression.(*ast.AttributeAccess)
	if !ok {
		t.Fatalf("expression is not *ast.AttributeAccess. got=%T", stmt.Expression)
	}

	if attr.Property == nil || attr.Property.Value != "name" {
		t.Fatalf("wrong final property. got=%v", attr.Property)
	}

	indexExpr, ok := attr.Object.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("attribute object is not *ast.IndexExpression. got=%T", attr.Object)
	}

	itemsAttr, ok := indexExpr.Left.(*ast.AttributeAccess)
	if !ok {
		t.Fatalf("index left is not *ast.AttributeAccess. got=%T", indexExpr.Left)
	}

	if itemsAttr.Property == nil || itemsAttr.Property.Value != "items" {
		t.Fatalf("wrong intermediate property. got=%v", itemsAttr.Property)
	}
}

func TestForDictWhereParses(t *testing.T) {
	input := `
for key, value in dict where value > 1 {
	value;
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("statement is not *ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	loop, ok := stmt.Expression.(*ast.ForEachMapLoop)
	if !ok {
		t.Fatalf("expression is not *ast.ForEachMapLoop. got=%T", stmt.Expression)
	}

	if loop.Key != "key" {
		t.Fatalf("wrong key variable. got=%q", loop.Key)
	}

	if loop.Value != "value" {
		t.Fatalf("wrong value variable. got=%q", loop.Value)
	}

	if loop.Cond == nil {
		t.Fatalf("expected where condition, got nil")
	}
}
