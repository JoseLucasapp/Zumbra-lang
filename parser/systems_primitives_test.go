package parser

import (
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
)

func TestSystemIntegerLiteralValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0xFF", 255},
		{"0b1010", 10},
		{"0o755", 493},
		{"1_000_000", 1_000_000},
	}

	for _, test := range tests {
		parser := New(lexer.New(test.input))
		program := parser.ParseProgram()
		checkParserErrors(t, parser)

		statement := program.Statements[0].(*ast.ExpressionStatement)
		literal := statement.Expression.(*ast.IntegerLiteral)
		if literal.Value != test.expected {
			t.Fatalf("%s: expected %d, got %d", test.input, test.expected, literal.Value)
		}
	}
}

func TestBitwiseOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 bor 2 bxor 3 band 4", "(1 bor (2 bxor (3 band 4)))"},
		{"1 + 2 shl 3", "((1 + 2) shl 3)"},
		{"1 shl 2 + 3", "(1 shl (2 + 3))"},
		{"bnot 1 band 0xFF", "((bnot1) band 0xFF)"},
		{"1 band 1 == 1", "((1 band 1) == 1)"},
		{"true and 1 bor 2 == 3", "(true and ((1 bor 2) == 3))"},
	}

	for _, test := range tests {
		parser := New(lexer.New(test.input))
		program := parser.ParseProgram()
		checkParserErrors(t, parser)

		if actual := program.String(); actual != test.expected {
			t.Fatalf("%s: expected %q, got %q", test.input, test.expected, actual)
		}
	}
}

func TestInvalidBaseLiteralReturnsParserError(t *testing.T) {
	parser := New(lexer.New("0b102"))
	_ = parser.ParseProgram()
	if len(parser.Errors()) == 0 {
		t.Fatal("expected parser error for invalid binary literal")
	}
}

func TestFloatLiteralWithSeparators(t *testing.T) {
	parser := New(lexer.New("10_000.25"))
	program := parser.ParseProgram()
	checkParserErrors(t, parser)

	statement := program.Statements[0].(*ast.ExpressionStatement)
	literal := statement.Expression.(*ast.FloatLiteral)
	if literal.Value != 10000.25 {
		t.Fatalf("expected 10000.25, got %f", literal.Value)
	}
}
