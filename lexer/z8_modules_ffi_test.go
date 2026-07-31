package lexer

import (
	"testing"
	"zumbra/token"
)

func TestZ8KeywordsAndArrow(t *testing.T) {
	input := `pub extern "C" from "native.c" { fct add(a: i32) -> i32; } unsafe { add(1i32); } import "m.zum" as m;`
	l := New(input)
	wanted := map[token.TokenType]bool{
		token.PUB: true, token.EXTERN: true, token.FROM: true,
		token.ARROW: true, token.UNSAFE: true, token.IMPORT: true, token.AS: true,
	}
	seen := map[token.TokenType]bool{}
	for {
		tok := l.NextToken()
		seen[tok.Type] = true
		if tok.Type == token.EOF {
			break
		}
	}
	for kind := range wanted {
		if !seen[kind] {
			t.Fatalf("token %s was not emitted", kind)
		}
	}
}
