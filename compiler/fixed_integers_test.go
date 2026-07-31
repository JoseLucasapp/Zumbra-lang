package compiler

import (
	"testing"

	"zumbra/object"
)

func TestCompileFixedIntegerConstants(t *testing.T) {
	program := parse("0xFFu8; 0xFFFFFFFFFFFFFFFFu64; 127i8")
	compiled := New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	constants := compiled.Bytecode().Constants
	if len(constants) != 3 {
		t.Fatalf("expected 3 constants, got %d", len(constants))
	}

	expected := []struct {
		kind object.FixedIntegerKind
		raw  uint64
	}{
		{object.FixedU8, 255},
		{object.FixedU64, ^uint64(0)},
		{object.FixedI8, 127},
	}

	for i, item := range expected {
		value, ok := constants[i].(*object.FixedInteger)
		if !ok {
			t.Fatalf("constant %d: expected fixed integer, got %T", i, constants[i])
		}
		if value.Kind != item.kind || value.UnsignedValue() != item.raw {
			t.Fatalf("constant %d: expected %s raw=%d, got %s raw=%d", i, item.kind, item.raw, value.Kind, value.UnsignedValue())
		}
	}
}
