package ast

import (
	"strconv"
	"zumbra/token"
)

type ImportStatement struct {
	Token token.Token
	Path  *StringLiteral
	Alias *Identifier
}

func (i *ImportStatement) statementNode()       {}
func (i *ImportStatement) TokenLiteral() string { return i.Token.Literal }

func (i *ImportStatement) String() string {
	if i.Path == nil {
		return "import;"
	}
	result := "import " + strconv.Quote(i.Path.Value)
	if i.Alias != nil {
		result += " as " + i.Alias.String()
	}
	return result + ";"
}
