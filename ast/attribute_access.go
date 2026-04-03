package ast

import "bytes"

type AttributeAccess struct {
	Object   Expression
	Property *Identifier
}

func (aa *AttributeAccess) expressionNode()      {}
func (aa *AttributeAccess) TokenLiteral() string { return aa.Object.TokenLiteral() }

func (aa *AttributeAccess) String() string {
	var out bytes.Buffer

	switch obj := aa.Object.(type) {
	case *IndexExpression:
		out.WriteString(obj.Left.String())
		out.WriteString("[")
		out.WriteString(obj.Index.String())
		out.WriteString("]")
	default:
		out.WriteString(aa.Object.String())
	}

	out.WriteString(".")
	out.WriteString(aa.Property.String())

	return out.String()
}
