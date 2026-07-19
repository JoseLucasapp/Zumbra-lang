package ast

import (
	"testing"
	"zumbra/token"
)

func TestIndexAssignStatementString(t *testing.T) {
	stmt := &IndexAssignStatement{
		Token: token.Token{Type: token.ASSIGN, Literal: "<<"},
		Target: &IndexExpression{
			Token: token.Token{Type: token.LBRACKET, Literal: "["},
			Left:  &Identifier{Token: token.Token{Type: token.IDENT, Literal: "memory"}, Value: "memory"},
			Index: &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "2"}, Value: 2},
		},
		Value: &IntegerLiteral{Token: token.Token{Type: token.INT, Literal: "169"}, Value: 169},
	}

	if got, want := stmt.String(), "(memory[2]) << 169;"; got != want {
		t.Fatalf("wrong string. want=%q got=%q", want, got)
	}
}
