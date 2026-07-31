package lexer

import (
	"testing"

	"zumbra/token"
)

func TestSystemIntegerLiteralsAndBitwiseKeywords(t *testing.T) {
	input := `0xFF 0b1010 0o755 1_000_000 10_000.25 bnot band bor bxor shl shr`

	tests := []struct {
		type_   token.TokenType
		literal string
	}{
		{token.INT, "0xFF"},
		{token.INT, "0b1010"},
		{token.INT, "0o755"},
		{token.INT, "1_000_000"},
		{token.FLOAT, "10_000.25"},
		{token.BIT_NOT, "bnot"},
		{token.BIT_AND, "band"},
		{token.BIT_OR, "bor"},
		{token.BIT_XOR, "bxor"},
		{token.SHIFT_L, "shl"},
		{token.SHIFT_R, "shr"},
		{token.EOF, ""},
	}

	lexer := New(input)
	for i, expected := range tests {
		actual := lexer.NextToken()
		if actual.Type != expected.type_ || actual.Literal != expected.literal {
			t.Fatalf(
				"token %d: expected (%s, %q), got (%s, %q)",
				i,
				expected.type_,
				expected.literal,
				actual.Type,
				actual.Literal,
			)
		}
	}
}

func TestInvalidBaseLiteralStaysWholeForParserDiagnostic(t *testing.T) {
	lexer := New("0b102")
	actual := lexer.NextToken()

	if actual.Type != token.INT {
		t.Fatalf("expected INT, got %s", actual.Type)
	}
	if actual.Literal != "0b102" {
		t.Fatalf("expected invalid literal to stay whole, got %q", actual.Literal)
	}
}
