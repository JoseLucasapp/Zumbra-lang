package ast

import (
	"testing"

	"zumbra/token"
)

func TestFixedIntegerLiteralKeepsTypeAndRawValue(t *testing.T) {
	literal := &IntegerLiteral{
		Token:     token.Token{Type: token.INT, Literal: "0xFFu8"},
		FixedType: "u8",
		RawValue:  255,
	}

	if literal.String() != "0xFFu8" {
		t.Fatalf("unexpected string: %s", literal.String())
	}
	if literal.FixedType != "u8" || literal.RawValue != 255 {
		t.Fatalf("unexpected fixed literal data: %s %d", literal.FixedType, literal.RawValue)
	}
}
