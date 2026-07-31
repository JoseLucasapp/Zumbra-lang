package ast

import (
	"bytes"
	"zumbra/token"
)

// SpawnExpression schedules a function call concurrently and returns a Task.
// The value must be a call expression so argument evaluation remains explicit
// and deterministic before the task starts.
type SpawnExpression struct {
	Token token.Token
	Value Expression
}

func (se *SpawnExpression) expressionNode()      {}
func (se *SpawnExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpawnExpression) String() string {
	var out bytes.Buffer
	out.WriteString(se.TokenLiteral())
	out.WriteString(" ")
	if se.Value != nil {
		out.WriteString(se.Value.String())
	}
	return out.String()
}
