package ast

import (
	"bytes"
	"zumbra/token"
)

type VarStatement struct {
	Token  token.Token
	Public bool
	Name   *Identifier
	Value  Expression
}

func (ls *VarStatement) statementNode()       {}
func (ls *VarStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *VarStatement) String() string {
	var out bytes.Buffer

	if ls.Public {
		out.WriteString("pub ")
	}
	out.WriteString(ls.TokenLiteral())
	out.WriteString(" ")
	out.WriteString(ls.Name.String())
	out.WriteString(" << ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}
