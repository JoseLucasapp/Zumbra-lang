package lexer

import (
	"testing"
	"zumbra/token"
)

func TestZ5Keywords(t *testing.T) {
	input := `const struct enum match case type`
	expected := []token.TokenType{token.CONST, token.STRUCT, token.ENUM, token.MATCH, token.CASE, token.TYPE, token.EOF}
	l := New(input)
	for index, want := range expected {
		got := l.NextToken()
		if got.Type != want {
			t.Fatalf("token %d: expected %s, got %s", index, want, got.Type)
		}
	}
}
