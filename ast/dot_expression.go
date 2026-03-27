package ast

import (
	"bytes"
	"zumbra/token"
)

type DotExpression struct {
	Token token.Token
	Left  Expression
	Right *Identifier
}

func (de *DotExpression) expressionNode()      {}
func (de *DotExpression) TokenLiteral() string { return de.Token.Literal }
func (de *DotExpression) String() string {
	var out bytes.Buffer
	if de.Left != nil {
		out.WriteString(de.Left.String())
	}
	out.WriteString(".")
	if de.Right != nil {
		out.WriteString(de.Right.String())
	}
	return out.String()
}
