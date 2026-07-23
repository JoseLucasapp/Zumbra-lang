package ast

import (
	"bytes"
	"strings"
	"zumbra/token"
)

type StructField struct {
	Token    token.Token
	Name     *Identifier
	TypeName string
}

type StructMethod struct {
	Token    token.Token
	Name     *Identifier
	Function *FunctionLiteral
}

// StructStatement declares a compact named data shape and its methods.
type StructStatement struct {
	Token       token.Token
	Public      bool
	Name        *Identifier
	Fields      []*StructField
	Methods     []*StructMethod
	RBraceToken token.Token
}

func (ss *StructStatement) statementNode()       {}
func (ss *StructStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *StructStatement) String() string {
	var out bytes.Buffer
	if ss.Public {
		out.WriteString("pub ")
	}
	out.WriteString("struct ")
	if ss.Name != nil {
		out.WriteString(ss.Name.String())
	}
	out.WriteString(" {")
	parts := []string{}
	for _, field := range ss.Fields {
		text := field.Name.String()
		if field.TypeName != "" {
			text += ": " + field.TypeName
		}
		parts = append(parts, text+";")
	}
	for _, method := range ss.Methods {
		parts = append(parts, "fct "+method.Name.String()+method.Function.String()[3:])
	}
	if len(parts) > 0 {
		out.WriteString(strings.Join(parts, " "))
	}
	out.WriteString("}")
	return out.String()
}
