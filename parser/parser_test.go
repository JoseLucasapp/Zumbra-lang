package parser

import (
	"testing"
	"../ast"
	"../lexer"
	"../token"
)

func TestVarStatements(t *testing.T) {
	input := `
		var x << 5;
		var y << 10;
		var foobar << 838383;
	`
	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()

	if program == nil {
		t.Fatalf("ParseProgram() returned nil")
	}

	if len(program.Statements) != 3 {
		t.Fatalf("program.Statements does not contain 3 statements. got=%d",
		len(program.Statements))
	}

	tests := []struct {
		expectedIdentifier string
	}{
		{"x"},
		{"y"},
		{"foobar"},
	}
	for i, tt := range tests {
		
		if program.Statements[i] == nil {
			t.Fatalf("Statement %d is nil", i)
		}

		stmt := program.Statements[i]
		if !testVarStatement(t, stmt, tt.expectedIdentifier) {
			return
		}
		
	}
}
func testVarStatement(t *testing.T, s ast.Statement, name string) bool {
	if s.TokenLiteral() != "var" {
		t.Errorf("s.TokenLiteral not 'var'. got=%q", s.TokenLiteral())
		return false
	}

	varStmt, ok := s.(*ast.VarStatement)
	if !ok {
		t.Errorf("s not *ast.VarStatement. got=%T", s)
		return false
	}

	if varStmt.Name.Value != name {
		t.Errorf("varStmt.Name.Value not '%s'. got=%s", name, varStmt.Name.Value)
		return false
	}
	if varStmt.Name.TokenLiteral() != name {
		t.Errorf("s.Name not '%s'. got=%s", name, varStmt.Name)
		return false
	}
	
	return true
}
func TestVarStatements2(t *testing.T) {
	input := `
		var x << 5;
		var y << 10;
		var foobar << 838383;
	`
	l := lexer.New(input)

	// Debug: veja se o "var" vira VAR mesmo
	for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
		t.Logf("TOKEN: Type=%s, Literal=%s", tok.Type, tok.Literal)
	}
}
