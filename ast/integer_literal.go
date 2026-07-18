package ast

import "zumbra/token"

type IntegerLiteral struct {
	Token token.Token

	// Value preserves the original untyped int behavior.
	Value int64

	// FixedType is empty for a normal int and contains u8/u16/u32/u64 or
	// i8/i16/i32/i64 for a fixed-width literal.
	FixedType string

	// RawValue stores the literal bits for fixed-width integers. It allows u64
	// literals to use the complete unsigned range without forcing them into int64.
	RawValue uint64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }
