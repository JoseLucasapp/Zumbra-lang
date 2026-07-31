package evaluator

import (
	"testing"
	"zumbra/object"
)

func TestEvaluateCompactMemoryCollections(t *testing.T) {
	result := testEval(`
        var memory << bytes(8);
        memory[2] << 0xA9u8;
        var window << slice(memory, 2, 5);
        window[1] << 0x42u8;
        memory[2] + memory[3];
    `)
	value, ok := result.(*object.FixedInteger)
	if !ok || value.Kind != object.FixedU8 || value.UnsignedValue() != 0xEB {
		t.Fatalf("unexpected result: %T %v", result, result)
	}
}

func TestEvaluateTypedArrayAndFill(t *testing.T) {
	result := testEval(`
        var registers << arrayOf("u16", 4);
        fill(registers, 0x10u16);
        registers[1] << 0xABCDu16;
        registers[1];
    `)
	value, ok := result.(*object.FixedInteger)
	if !ok || value.Kind != object.FixedU16 || value.UnsignedValue() != 0xABCD {
		t.Fatalf("unexpected result: %T %v", result, result)
	}
}
