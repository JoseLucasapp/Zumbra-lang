package ast

import (
	"bytes"
	"strconv"
	"strings"
	"zumbra/token"
)

// ExternType describes a C-compatible scalar, pointer, string or callback type.
type ExternType struct {
	Name           string
	CallbackParams []*ExternType
	CallbackReturn *ExternType
}

func (t *ExternType) String() string {
	if t == nil {
		return "void"
	}
	if t.Name != "callback" {
		return t.Name
	}
	parts := make([]string, 0, len(t.CallbackParams))
	for _, param := range t.CallbackParams {
		parts = append(parts, param.String())
	}
	result := "callback(" + strings.Join(parts, ", ") + ")"
	if t.CallbackReturn != nil {
		result += " -> " + t.CallbackReturn.String()
	}
	return result
}

type ExternParam struct {
	Token token.Token
	Name  *Identifier
	Type  *ExternType
}

type ExternFunction struct {
	Token      token.Token
	Name       *Identifier
	CName      string
	Parameters []*ExternParam
	ReturnType *ExternType
}

// ExternBlockStatement declares functions supplied by a C library or source.
type ExternBlockStatement struct {
	Token     token.Token
	Public    bool
	ABI       string
	Link      string
	Functions []*ExternFunction
}

func (s *ExternBlockStatement) statementNode()       {}
func (s *ExternBlockStatement) TokenLiteral() string { return s.Token.Literal }
func (s *ExternBlockStatement) String() string {
	var out bytes.Buffer
	if s.Public {
		out.WriteString("pub ")
	}
	out.WriteString("extern ")
	out.WriteString(strconv.Quote(s.ABI))
	if s.Link != "" {
		out.WriteString(" from ")
		out.WriteString(strconv.Quote(s.Link))
	}
	out.WriteString(" {")
	for _, fn := range s.Functions {
		out.WriteString(" fct ")
		out.WriteString(fn.Name.String())
		out.WriteString("(")
		params := make([]string, 0, len(fn.Parameters))
		for _, param := range fn.Parameters {
			params = append(params, param.Name.String()+": "+param.Type.String())
		}
		out.WriteString(strings.Join(params, ", "))
		out.WriteString(") -> ")
		out.WriteString(fn.ReturnType.String())
		if fn.CName != "" && fn.Name != nil && fn.CName != fn.Name.Value {
			out.WriteString(" as ")
			out.WriteString(strconv.Quote(fn.CName))
		}
		out.WriteString(";")
	}
	out.WriteString(" }")
	return out.String()
}
