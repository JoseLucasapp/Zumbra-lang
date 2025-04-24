package ast

import "zumbra/token"

type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (lf *FloatLiteral) expressionNode()      {}
func (lf *FloatLiteral) TokenLiteral() string { return lf.Token.Literal }
func (lf *FloatLiteral) String() string       { return lf.Token.Literal }
