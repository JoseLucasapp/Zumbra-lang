package lexer

import (
	"testing"
	"zumbra/token"
)

func TestSpawnKeyword(t *testing.T) {
	l := New("spawn work();")
	if got := l.NextToken(); got.Type != token.SPAWN {
		t.Fatalf("expected SPAWN, got %s", got.Type)
	}
}
