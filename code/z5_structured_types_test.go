package code

import "testing"

func TestZ5OpcodeDefinitions(t *testing.T) {
	for _, op := range []Opcode{OpStructDefinition, OpEnumDefinition, OpSetAttr} {
		if _, err := Lookup(byte(op)); err != nil {
			t.Fatalf("missing opcode %d: %v", op, err)
		}
	}
	if got := Make(OpStructDefinition, 2, 1); len(got) != 3 {
		t.Fatalf("unexpected instruction length: %d", len(got))
	}
}
