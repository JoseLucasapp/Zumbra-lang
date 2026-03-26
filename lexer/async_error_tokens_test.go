package lexer

import (
	"testing"
	"zumbra/token"
)

func TestAsyncAwaitTryOrTokens(t *testing.T) {
	input := `
async fct(x) {
    return;
}

await task;
try call();
result or {
    return;
}
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.ASYNC, "async"},
		{token.FUNCTION, "fct"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.AWAIT, "await"},
		{token.IDENT, "task"},
		{token.SEMICOLON, ";"},
		{token.TRY, "try"},
		{token.IDENT, "call"},
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.IDENT, "result"},
		{token.OR, "or"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - token type wrong. expected=%q, got=%q", i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}
