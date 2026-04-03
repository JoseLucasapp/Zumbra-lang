package ast

import (
	"bytes"
	"zumbra/token"
)

type AwaitExpression struct {
	Token token.Token
	Value Expression
}

func (ae *AwaitExpression) expressionNode()      {}
func (ae *AwaitExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AwaitExpression) String() string {
	var out bytes.Buffer
	out.WriteString(ae.TokenLiteral())
	out.WriteString(" ")

	switch v := ae.Value.(type) {
	case *IndexExpression:
		out.WriteString(v.Left.String())
		out.WriteString("[")
		out.WriteString(v.Index.String())
		out.WriteString("]")
	default:
		if ae.Value != nil {
			out.WriteString(ae.Value.String())
		}
	}

	return out.String()
}
