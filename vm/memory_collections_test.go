package vm

import (
	"testing"
	"zumbra/compiler"
	"zumbra/object"
)

func TestVMCompactMemoryCollections(t *testing.T) {
	tests := []struct {
		input string
		kind  object.FixedIntegerKind
		want  uint64
	}{
		{`var memory << bytes(4); memory[1] << 0xA9u8; memory[1];`, object.FixedU8, 0xA9},
		{`var regs << arrayOf("u16", 2); regs[0] << 0x1234u16; regs[0];`, object.FixedU16, 0x1234},
		{`var memory << bytes(4); var part << slice(memory, 1, 3); part[0] << 0x44u8; memory[1];`, object.FixedU8, 0x44},
	}

	for _, test := range tests {
		program := parse(test.input)
		compiled := compiler.New()
		if err := compiled.Compile(program); err != nil {
			t.Fatalf("compiler error: %v", err)
		}
		machine := New(compiled.Bytecode())
		if err := machine.Run(); err != nil {
			t.Fatalf("VM error: %v", err)
		}
		actual, ok := machine.LastPoppedStackElem().(*object.FixedInteger)
		if !ok {
			t.Fatalf("expected fixed integer, got %T", machine.LastPoppedStackElem())
		}
		if actual.Kind != test.kind || actual.UnsignedValue() != test.want {
			t.Fatalf("expected %s/%d, got %s/%d", test.kind, test.want, actual.Kind, actual.UnsignedValue())
		}
	}
}
