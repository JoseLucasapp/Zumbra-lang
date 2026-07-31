package ast

import (
	"bytes"
	"zumbra/token"
)

// UnsafeStatement makes machine-level or FFI operations explicit.
type UnsafeStatement struct {
	Token token.Token
	Body  *BlockStatement
}

func (s *UnsafeStatement) statementNode()       {}
func (s *UnsafeStatement) TokenLiteral() string { return s.Token.Literal }
func (s *UnsafeStatement) String() string {
	var out bytes.Buffer
	out.WriteString("unsafe ")
	if s.Body != nil {
		out.WriteString(s.Body.String())
	}
	return out.String()
}
