package ast

import (
	"bytes"
	"zumbra/token"
)

// ConstStatement declares an immutable value.
type ConstStatement struct {
	Token  token.Token
	Public bool
	Name   *Identifier
	Value  Expression
}

func (cs *ConstStatement) statementNode()       {}
func (cs *ConstStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ConstStatement) String() string {
	var out bytes.Buffer
	if cs.Public {
		out.WriteString("pub ")
	}
	out.WriteString("const ")
	if cs.Name != nil {
		out.WriteString(cs.Name.String())
	}
	out.WriteString(" << ")
	if cs.Value != nil {
		out.WriteString(cs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}
