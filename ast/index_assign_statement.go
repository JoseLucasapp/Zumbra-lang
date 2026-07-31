package ast

import (
	"bytes"
	"zumbra/token"
)

// IndexAssignStatement mutates one element of an indexable value.
// Example: memory[0x200] << 0xA9u8;
type IndexAssignStatement struct {
	Token  token.Token
	Target *IndexExpression
	Value  Expression
}

func (ias *IndexAssignStatement) statementNode()       {}
func (ias *IndexAssignStatement) TokenLiteral() string { return ias.Token.Literal }
func (ias *IndexAssignStatement) String() string {
	var out bytes.Buffer

	if ias.Target != nil {
		out.WriteString(ias.Target.String())
	}
	out.WriteString(" << ")
	if ias.Value != nil {
		out.WriteString(ias.Value.String())
	}
	out.WriteString(";")

	return out.String()
}
