package lexer

import (
	"testing"

	"zumbra/token"
)

func TestFixedIntegerSuffixesStayInIntegerToken(t *testing.T) {
	input := `255u8 65_535u16 0xFFFFu16 0b1111u8 0o77u8 10i8 20i16 30i32 40i64 0xFFFFFFFFFFFFFFFFu64`
	expected := []string{
		"255u8", "65_535u16", "0xFFFFu16", "0b1111u8", "0o77u8",
		"10i8", "20i16", "30i32", "40i64", "0xFFFFFFFFFFFFFFFFu64",
	}

	lexer := New(input)
	for i, literal := range expected {
		actual := lexer.NextToken()
		if actual.Type != token.INT || actual.Literal != literal {
			t.Fatalf("token %d: expected INT %q, got %s %q", i, literal, actual.Type, actual.Literal)
		}
	}
}

func TestUnknownIntegerSuffixIsNotConsumed(t *testing.T) {
	lexer := New("255u7")
	integer := lexer.NextToken()
	suffix := lexer.NextToken()

	if integer.Type != token.INT || integer.Literal != "255" {
		t.Fatalf("expected INT 255, got %s %q", integer.Type, integer.Literal)
	}
	if suffix.Type != token.IDENT || suffix.Literal != "u7" {
		t.Fatalf("expected IDENT u7, got %s %q", suffix.Type, suffix.Literal)
	}
}
