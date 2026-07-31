package semantic

import (
	"fmt"
	"zumbra/token"
)

func formatTokenPosition(tok token.Token) string {
	if tok.Pos.IsValid() {
		return fmt.Sprintf(" at line %d, col %d", tok.Pos.Line, tok.Pos.Col)
	}
	return ""
}

func ErrDuplicateSymbol(name string) error {
	return fmt.Errorf("symbol already declared in this scope: %s", name)
}

func ErrDuplicateSymbolAt(name string, tok token.Token) error {
	return fmt.Errorf("symbol already declared in this scope: %s%s", name, formatTokenPosition(tok))
}

func ErrUndefinedSymbol(name string) error {
	return fmt.Errorf("undefined symbol: %s", name)
}

func ErrUndefinedSymbolAt(name string, tok token.Token) error {
	return fmt.Errorf("undefined symbol: %s%s", name, formatTokenPosition(tok))
}

func ErrAssignmentToUndefinedSymbol(name string) error {
	return fmt.Errorf("assignment to undefined symbol: %s", name)
}

func ErrAssignmentToUndefinedSymbolAt(name string, tok token.Token) error {
	return fmt.Errorf("assignment to undefined symbol: %s%s", name, formatTokenPosition(tok))
}

func ErrAssignmentToImmutableSymbolAt(name string, tok token.Token) error {
	return fmt.Errorf("cannot assign to immutable symbol: %s%s", name, formatTokenPosition(tok))
}
