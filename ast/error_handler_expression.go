package ast

import (
	"bytes"
	"zumbra/token"
)

type ErrorHandlerExpression struct {
	Token      token.Token
	Left       Expression
	ErrorIdent *Identifier
	Handler    *BlockStatement
}

func (ehe *ErrorHandlerExpression) expressionNode()      {}
func (ehe *ErrorHandlerExpression) TokenLiteral() string { return ehe.Token.Literal }

func (ehe *ErrorHandlerExpression) String() string {
	var out bytes.Buffer

	if ehe.Left != nil {
		out.WriteString(ehe.Left.String())
		out.WriteString(" ")
	}

	out.WriteString("or")

	if ehe.ErrorIdent != nil {
		out.WriteString(" ")
		out.WriteString(ehe.ErrorIdent.String())
	}

	out.WriteString(" ")

	out.WriteString("{")
	if ehe.Handler != nil {
		out.WriteString(ehe.Handler.String())
	}
	out.WriteString("}")

	return out.String()
}
