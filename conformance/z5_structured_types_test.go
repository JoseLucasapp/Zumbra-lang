package conformance

import "testing"

func TestZ5MatchesEvaluatorAndVM(t *testing.T) {
	sources := []string{
		`struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } } var p << Point(1, 2); p.move(3, 4); p.x + p.y;`,
		`enum Direction { Up; Down; } match(Direction.Down) { case Direction.Up { 1; } case Direction.Down { 2; } else { 0; } };`,
		`struct Cpu { a: u8; pc: u16; } var cpu << Cpu({"a": 0xA9u8, "pc": 0x8000u16}); cpu.pc;`,
	}
	for _, source := range sources {
		evaluatorResult := indexAssignmentEvaluatorResult(t, source)
		vmResult := indexAssignmentVMResult(t, source)
		if evaluatorResult.Type() != vmResult.Type() || evaluatorResult.Inspect() != vmResult.Inspect() {
			t.Fatalf("mismatch: evaluator=%s vm=%s", evaluatorResult.Inspect(), vmResult.Inspect())
		}
	}
}
