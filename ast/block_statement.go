package ast

import (
	"bytes"
	"zumbra/token"
)

type BlockStatement struct {
	Token       token.Token
	Statements  []Statement
	RBraceToken token.Token
}

func (bs *BlockStatement) Pos() token.Position {
	return bs.Token.Pos
}

func (bs *BlockStatement) End() token.Position {
	return token.Position{
		Filename: bs.RBraceToken.Pos.Filename,
		Line:     bs.RBraceToken.Pos.Line,
		Col:      bs.RBraceToken.Pos.Col + 1,
	}
}

func (bs *BlockStatement) expressionNode()      {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }

func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		str := s.String()
		out.WriteString(str)

		if str == "" {
			continue
		}

		last := str[len(str)-1]
		if last != ';' && last != '}' {
			out.WriteString(";")
		}
	}

	return out.String()
}
