package vm

import (
	"testing"
	"zumbra/compiler"
	"zumbra/object"
	"zumbra/object/builtins"
)

func TestVMSupportsBuiltinIndexesAboveUint8(t *testing.T) {
	var builtinIndex int = -1
	for i, builtin := range builtins.Builtins {
		if builtin.Name == "bytesEqual" {
			builtinIndex = i
			break
		}
	}
	if builtinIndex <= 255 {
		t.Fatalf("regression requires bytesEqual above the uint8 boundary, got index %d", builtinIndex)
	}

	program := parse(`
        var left << bytes(2);
        left[0] << 0x5Au8;
        var right << bytes(2);
        copyBytes(right, 0, left, 0, 2);
        bytesEqual(left, right);
    `)

	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	machine := New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("VM error: %v", err)
	}

	result, ok := machine.LastPoppedStackElem().(*object.Boolean)
	if !ok || !result.Value {
		t.Fatalf("expected true from builtin index %d, got %T %v", builtinIndex, machine.LastPoppedStackElem(), machine.LastPoppedStackElem())
	}
}
