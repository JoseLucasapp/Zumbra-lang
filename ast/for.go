package ast

import (
	"bytes"
	"zumbra/token"
)

type ForLoop struct {
	Token  token.Token
	Init   Expression
	Cond   Expression
	Update Expression
	Block  *BlockStatement
}

func (fl *ForLoop) Pos() token.Position {
	return fl.Token.Pos
}

func (fl *ForLoop) End() token.Position {
	return fl.Block.End()
}

func (fl *ForLoop) expressionNode()      {}
func (fl *ForLoop) TokenLiteral() string { return fl.Token.Literal }

func (fl *ForLoop) String() string {
	var out bytes.Buffer

	out.WriteString("for")
	out.WriteString(" ( ")
	out.WriteString(fl.Init.String())
	out.WriteString(" ; ")
	out.WriteString(fl.Cond.String())
	out.WriteString(" ; ")
	out.WriteString(fl.Update.String())
	out.WriteString(" ) ")
	out.WriteString(" { ")
	out.WriteString(fl.Block.String())
	out.WriteString(" }")

	return out.String()
}

type ForEachArrayLoop struct {
	Token token.Token
	Var   string
	Value Expression //value to range over
	Cond  Expression //conditional clause(nil if there is no 'WHERE' clause)
	Block *BlockStatement
}

func (fal *ForEachArrayLoop) Pos() token.Position {
	return fal.Token.Pos
}

func (fal *ForEachArrayLoop) End() token.Position {
	return fal.Block.End()
}

func (fal *ForEachArrayLoop) expressionNode()      {}
func (fal *ForEachArrayLoop) TokenLiteral() string { return fal.Token.Literal }

func (fal *ForEachArrayLoop) String() string {
	var out bytes.Buffer

	out.WriteString("for ")
	out.WriteString(fal.Var)
	out.WriteString(" in ")
	out.WriteString(fal.Value.String())
	if fal.Cond != nil {
		out.WriteString(" where ")
		out.WriteString(fal.Cond.String())
	}
	out.WriteString(" { ")
	out.WriteString(fal.Block.String())
	out.WriteString(" }")

	return out.String()
}

type ForEachMapLoop struct {
	Token token.Token
	Key   string
	Value string
	X     Expression //value to range over
	Cond  Expression //Conditional clause(nil if there is no 'WHERE' clause)
	Block *BlockStatement
}

func (fml *ForEachMapLoop) Pos() token.Position {
	return fml.Token.Pos
}

func (fml *ForEachMapLoop) End() token.Position {
	return fml.Block.End()
}

func (fml *ForEachMapLoop) expressionNode()      {}
func (fml *ForEachMapLoop) TokenLiteral() string { return fml.Token.Literal }

func (fml *ForEachMapLoop) String() string {
	var out bytes.Buffer

	out.WriteString("for ")
	out.WriteString(fml.Key + ", " + fml.Value)
	out.WriteString(" in ")
	out.WriteString(fml.X.String())
	if fml.Cond != nil {
		out.WriteString(" where ")
		out.WriteString(fml.Cond.String())
	}
	out.WriteString(" { ")
	out.WriteString(fml.Block.String())
	out.WriteString(" }")

	return out.String()
}

type ForEverLoop struct {
	Token token.Token
	Block *BlockStatement
}

func (fel *ForEverLoop) Pos() token.Position {
	return fel.Token.Pos
}

func (fel *ForEverLoop) End() token.Position {
	return fel.Block.End()
}

func (fel *ForEverLoop) expressionNode()      {}
func (fel *ForEverLoop) TokenLiteral() string { return fel.Token.Literal }

func (fel *ForEverLoop) String() string {
	var out bytes.Buffer

	out.WriteString("for ")
	out.WriteString(" { ")
	out.WriteString(fel.Block.String())
	out.WriteString(" }")

	return out.String()
}

// for i in start..end <where cond> { }
type ForEachDotRange struct {
	Token    token.Token
	Var      string
	StartIdx Expression
	EndIdx   Expression
	Cond     Expression //conditional clause(nil if there is no 'WHERE' clause)
	Block    *BlockStatement
}

func (fdr *ForEachDotRange) Pos() token.Position {
	return fdr.Token.Pos
}

func (fdr *ForEachDotRange) End() token.Position {
	return fdr.Block.End()
}

func (fdr *ForEachDotRange) expressionNode()      {}
func (fdr *ForEachDotRange) TokenLiteral() string { return fdr.Token.Literal }

func (fdr *ForEachDotRange) String() string {
	var out bytes.Buffer

	out.WriteString("for ")
	out.WriteString(fdr.Var)
	out.WriteString(" in ")
	out.WriteString(fdr.StartIdx.String())
	out.WriteString(" .. ")
	out.WriteString(fdr.EndIdx.String())
	if fdr.Cond != nil {
		out.WriteString(" where ")
		out.WriteString(fdr.Cond.String())
	}
	out.WriteString(" { ")
	out.WriteString(fdr.Block.String())
	out.WriteString(" }")

	return out.String()
}
