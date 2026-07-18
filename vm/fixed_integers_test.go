package vm

import (
	"strings"
	"testing"

	"zumbra/compiler"
	"zumbra/object"
)

func TestFixedIntegersOnVM(t *testing.T) {
	tests := []struct {
		input    string
		kind     object.FixedIntegerKind
		expected uint64
	}{
		{"255u8 + 1", object.FixedU8, 0},
		{"0u8 - 1", object.FixedU8, 255},
		{"bnot 0u8", object.FixedU8, 255},
		{"255u8 shl 1", object.FixedU8, 254},
		{"u16(0x8000)", object.FixedU16, 32768},
		{"satAdd(255u8, 1u8)", object.FixedU8, 255},
	}

	for _, test := range tests {
		program := parse(test.input)
		compiled := compiler.New()
		if err := compiled.Compile(program); err != nil {
			t.Fatalf("%s: compiler error: %v", test.input, err)
		}
		machine := New(compiled.Bytecode())
		if err := machine.Run(); err != nil {
			t.Fatalf("%s: VM error: %v", test.input, err)
		}
		value, ok := machine.LastPoppedStackElem().(*object.FixedInteger)
		if !ok {
			t.Fatalf("%s: expected fixed integer, got %T", test.input, machine.LastPoppedStackElem())
		}
		if value.Kind != test.kind || value.UnsignedValue() != test.expected {
			t.Fatalf("%s: expected %s raw=%d, got %s raw=%d", test.input, test.kind, test.expected, value.Kind, value.UnsignedValue())
		}
	}
}

func TestCheckedOverflowOnVMReturnsErrorObject(t *testing.T) {
	program := parse("checkedAdd(255u8, 1u8)")
	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}
	machine := New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("VM error: %v", err)
	}
	value, ok := machine.LastPoppedStackElem().(*object.Error)
	if !ok {
		t.Fatalf("expected error object, got %T", machine.LastPoppedStackElem())
	}
	if !strings.Contains(value.Message, "u8 overflow") {
		t.Fatalf("unexpected error: %s", value.Message)
	}
}
