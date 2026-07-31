package ast

import (
	"bytes"
	"zumbra/token"
)

// AttributeAssignStatement mutates a field such as player.x << 10.
type AttributeAssignStatement struct {
	Token  token.Token
	Target *AttributeAccess
	Value  Expression
}

func (as *AttributeAssignStatement) statementNode()       {}
func (as *AttributeAssignStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AttributeAssignStatement) String() string {
	var out bytes.Buffer
	if as.Target != nil {
		out.WriteString(as.Target.String())
	}
	out.WriteString(" << ")
	if as.Value != nil {
		out.WriteString(as.Value.String())
	}
	out.WriteString(";")
	return out.String()
}
