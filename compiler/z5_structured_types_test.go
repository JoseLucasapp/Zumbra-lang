package compiler

import (
	"strings"
	"testing"
	"zumbra/code"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestCompileZ5StructuredTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } }
        enum Direction { Up; Down; }
        var p << Point(1, 2);
        p.x << 3;
        match(Direction.Up) { case Direction.Up { p.x; } else { 0; } };
    `))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}
	c := New()
	if err := c.Compile(program); err != nil {
		t.Fatal(err)
	}
	text := c.Bytecode().Instructions.String()
	for _, expected := range []string{"OpStructDefinition", "OpEnumDefinition", "OpSetAttr", "OpGetAttr"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %s\n%s", expected, text)
		}
	}
}

func TestCompileRejectsConstantAssignment(t *testing.T) {
	p := parser.New(lexer.New(`const Max << 3; Max << 4;`))
	c := New()
	err := c.Compile(p.ParseProgram())
	if err == nil || !strings.Contains(err.Error(), "cannot assign to constant") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStructOpcodeEncoding(t *testing.T) {
	if got := code.Make(code.OpStructDefinition, 2, 1); len(got) != 3 {
		t.Fatalf("unexpected opcode: %v", got)
	}
}
