package parser

import (
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
)

func TestParseFixedIntegerLiterals(t *testing.T) {
	tests := []struct {
		input     string
		fixedType string
		raw       uint64
	}{
		{"255u8", "u8", 255},
		{"0xFFFFu16", "u16", 65535},
		{"0b1010u32", "u32", 10},
		{"0xFFFFFFFFFFFFFFFFu64", "u64", ^uint64(0)},
		{"127i8", "i8", 127},
		{"32767i16", "i16", 32767},
		{"2147483647i32", "i32", 2147483647},
		{"9223372036854775807i64", "i64", 9223372036854775807},
	}

	for _, test := range tests {
		parser := New(lexer.New(test.input))
		program := parser.ParseProgram()
		checkParserErrors(t, parser)

		statement := program.Statements[0].(*ast.ExpressionStatement)
		literal := statement.Expression.(*ast.IntegerLiteral)
		if literal.FixedType != test.fixedType || literal.RawValue != test.raw {
			t.Fatalf("%s: expected %s raw=%d, got %s raw=%d", test.input, test.fixedType, test.raw, literal.FixedType, literal.RawValue)
		}
	}
}

func TestFixedIntegerLiteralRangeValidation(t *testing.T) {
	for _, input := range []string{"256u8", "128i8", "65536u16", "0x1_0000_0000u32"} {
		parser := New(lexer.New(input))
		_ = parser.ParseProgram()
		if len(parser.Errors()) == 0 {
			t.Fatalf("expected range error for %s", input)
		}
	}
}
