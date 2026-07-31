package ast

import (
	"bytes"
	"zumbra/token"
)

type TryExpression struct {
	Token token.Token
	Value Expression
}

func (te *TryExpression) expressionNode()      {}
func (te *TryExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TryExpression) String() string {
	var out bytes.Buffer
	out.WriteString(te.TokenLiteral())
	out.WriteString(" ")

	switch v := te.Value.(type) {
	case *IndexExpression:
		out.WriteString(v.Left.String())
		out.WriteString("[")
		out.WriteString(v.Index.String())
		out.WriteString("]")
	default:
		if te.Value != nil {
			out.WriteString(te.Value.String())
		}
	}

	return out.String()
}
