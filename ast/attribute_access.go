package ast

import (
	"bytes"
)

type AttributeAccess struct {
	Object   Expression
	Property *Identifier
}

func (aa *AttributeAccess) expressionNode()      {}
func (aa *AttributeAccess) TokenLiteral() string { return aa.Object.TokenLiteral() }
func (aa *AttributeAccess) String() string {
	var out bytes.Buffer

	out.WriteString(aa.Object.String())
	out.WriteString(".")
	out.WriteString(aa.Property.String())

	return out.String()
}
