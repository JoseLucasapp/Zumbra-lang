package vm

import "testing"

func TestVMZ5StructMethodsEnumsAndMatch(t *testing.T) {
	tests := []vmTestCase{{
		input: `
            struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } }
            enum Direction { Up; Down; }
            var p << Point(1, 2);
            p.move(3, 4);
            p.x << p.x + 1;
            match(Direction.Up) { case Direction.Up { p.x + p.y; } else { 0; } };
        `,
		expected: 11,
	}}
	runVmTests(t, tests)
}

func TestVMZ5NamedStructConstruction(t *testing.T) {
	tests := []vmTestCase{{
		input:    `struct Cpu { opcode: u8; pc: u16; } var cpu << Cpu({"pc": 0x8000u16, "opcode": 0xA9u8}); cpu.opcode;`,
		expected: uint8(169),
	}}
	runVmTests(t, tests)
}
