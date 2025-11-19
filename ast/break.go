package ast

import (
	"unicode/utf8"
	"zumbra/token"
)

type BreakExpression struct {
	Token token.Token
}

func (be *BreakExpression) Pos() token.Position {
	return be.Token.Pos
}

func (be *BreakExpression) End() token.Position {
	length := utf8.RuneCountInString(be.Token.Literal)
	pos := be.Token.Pos
	return token.Position{Filename: pos.Filename, Line: pos.Line, Col: pos.Col + length}
}

func (be *BreakExpression) expressionNode()      {}
func (be *BreakExpression) TokenLiteral() string { return be.Token.Literal }

func (be *BreakExpression) String() string { return be.Token.Literal }
