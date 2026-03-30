package ast

import "zumbra/token"

type ImportStatement struct {
	Token token.Token
	Path  *StringLiteral
}

func (i *ImportStatement) statementNode()       {}
func (i *ImportStatement) TokenLiteral() string { return i.Token.Literal }

func (i *ImportStatement) String() string {
	if i.Path == nil {
		return "import;"
	}
	return "import " + i.Path.String() + ";"
}
