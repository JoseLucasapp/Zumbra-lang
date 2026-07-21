package ast

import (
	"bytes"
	"zumbra/token"
)

// TypeAliasStatement gives a simpler name to an existing type.
type TypeAliasStatement struct {
	Token  token.Token
	Name   *Identifier
	Target *Identifier
}

func (ts *TypeAliasStatement) statementNode()       {}
func (ts *TypeAliasStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *TypeAliasStatement) String() string {
	var out bytes.Buffer
	out.WriteString("type ")
	if ts.Name != nil {
		out.WriteString(ts.Name.String())
	}
	out.WriteString(" << ")
	if ts.Target != nil {
		out.WriteString(ts.Target.String())
	}
	out.WriteString(";")
	return out.String()
}
