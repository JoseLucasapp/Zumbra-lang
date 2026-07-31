package ast

import (
	"bytes"
	"zumbra/token"
)

type MatchCase struct {
	Token   token.Token
	Pattern Expression
	Body    *BlockStatement
}

// MatchExpression chooses the first case equal to Value.
type MatchExpression struct {
	Token       token.Token
	Value       Expression
	Cases       []*MatchCase
	Default     *BlockStatement
	RBraceToken token.Token
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) String() string {
	var out bytes.Buffer
	out.WriteString("match(")
	if me.Value != nil {
		out.WriteString(me.Value.String())
	}
	out.WriteString(") {")
	for _, c := range me.Cases {
		out.WriteString("case ")
		if c.Pattern != nil {
			out.WriteString(c.Pattern.String())
		}
		out.WriteString(" {")
		if c.Body != nil {
			out.WriteString(c.Body.String())
		}
		out.WriteString("}")
	}
	if me.Default != nil {
		out.WriteString("else {")
		out.WriteString(me.Default.String())
		out.WriteString("}")
	}
	out.WriteString("}")
	return out.String()
}
