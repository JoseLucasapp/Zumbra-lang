package semantic

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestResolveZ5Program(t *testing.T) {
	p := parser.New(lexer.New(`
        const Max << 3;
        type Byte << u8;
        struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; } }
        enum Direction { Up; Down; }
        var p << Point(1, 2);
        p.move(1, 2);
        match(Direction.Up) { case Direction.Up { p.x; } else { 0; } };
    `))
	r := NewResolver()
	if errors := r.Resolve(p.ParseProgram()); len(errors) != 0 {
		t.Fatalf("resolver errors: %v", errors)
	}
}

func TestResolveRejectsConstantAssignment(t *testing.T) {
	p := parser.New(lexer.New(`const Max << 3; Max << 4;`))
	errors := NewResolver().Resolve(p.ParseProgram())
	if len(errors) == 0 || !strings.Contains(errors[0].Error(), "immutable") {
		t.Fatalf("unexpected errors: %v", errors)
	}
}

func TestStructMethodsDoNotProduceUnusedFunctionWarnings(t *testing.T) {
	p := parser.New(lexer.New(`
        struct Player {
            x: int;
            fct move(amount) { self.x << self.x + amount; }
        }
        var player << Player(0);
        player.move(1);
    `))
	resolver := NewResolver()
	if errors := resolver.Resolve(p.ParseProgram()); len(errors) != 0 {
		t.Fatalf("resolver errors: %v", errors)
	}
	for _, warning := range resolver.Result().Warnings {
		if strings.Contains(warning.Message, "unused function: move") {
			t.Fatalf("unexpected method warning: %v", warning)
		}
	}
}
