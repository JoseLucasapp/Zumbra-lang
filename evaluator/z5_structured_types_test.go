package evaluator

import (
	"strings"
	"testing"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/parser"
)

func evalZ5(t *testing.T, input string) object.Object {
	t.Helper()
	p := parser.New(lexer.New(input))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	return Eval(program, object.NewEnvironment())
}

func TestZ5StructMethodsEnumsAndMatch(t *testing.T) {
	result := evalZ5(t, `
        struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } }
        enum Direction { Up; Down; }
        var p << Point(1, 2);
        p.move(3, 4);
        p.x << p.x + 1;
        match(Direction.Up) { case Direction.Up { p.x + p.y; } else { 0; } };
    `)
	if result.Inspect() != "11" {
		t.Fatalf("expected 11, got %s", result.Inspect())
	}
}

func TestZ5ConstantCannotBeAssigned(t *testing.T) {
	result := evalZ5(t, `const Max << 3; Max << 4;`)
	if result.Type() != object.ERROR_OBJ || !strings.Contains(result.Inspect(), "cannot assign to constant") {
		t.Fatalf("unexpected result: %s", result.Inspect())
	}
}

func TestZ5NamedStructConstruction(t *testing.T) {
	result := evalZ5(t, `
        struct Cpu { opcode: u8; pc: u16; }
        var cpu << Cpu({"pc": 0x8000u16, "opcode": 0xA9u8});
        cpu.opcode;
    `)
	if result.Inspect() != "169" {
		t.Fatalf("expected 169, got %s", result.Inspect())
	}
}

func TestZ5RejectsUnknownStructField(t *testing.T) {
	result := evalZ5(t, `struct Point { x: int; } var p << Point(1); p.y;`)
	if result.Type() != object.ERROR_OBJ || !strings.Contains(result.Inspect(), "unknown field") {
		t.Fatalf("unexpected result: %s", result.Inspect())
	}
}
