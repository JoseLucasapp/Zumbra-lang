package ast

import (
	"bytes"
	"strings"
	"zumbra/token"
)

// EnumStatement declares a finite set of named values.
type EnumStatement struct {
	Token       token.Token
	Public      bool
	Name        *Identifier
	Members     []*Identifier
	RBraceToken token.Token
}

func (es *EnumStatement) statementNode()       {}
func (es *EnumStatement) TokenLiteral() string { return es.Token.Literal }
func (es *EnumStatement) String() string {
	var out bytes.Buffer
	if es.Public {
		out.WriteString("pub ")
	}
	out.WriteString("enum ")
	if es.Name != nil {
		out.WriteString(es.Name.String())
	}
	out.WriteString(" {")
	members := []string{}
	for _, member := range es.Members {
		members = append(members, member.String())
	}
	out.WriteString(strings.Join(members, "; "))
	out.WriteString("}")
	return out.String()
}
