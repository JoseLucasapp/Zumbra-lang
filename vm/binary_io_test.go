package vm

import (
	"fmt"
	"path/filepath"
	"testing"
	"zumbra/compiler"
	"zumbra/object"
)

func TestVMBinaryEndianCopyAndHash(t *testing.T) {
	tests := []struct {
		input string
		check func(object.Object) bool
	}{
		{
			`var data << bytes(8); writeU32LE(data, 0, 0x78563412u32); readU32LE(data, 0);`,
			func(value object.Object) bool {
				fixed, ok := value.(*object.FixedInteger)
				return ok && fixed.Kind == object.FixedU32 && fixed.UnsignedValue() == 0x78563412
			},
		},
		{
			`var source << bytes(3); source[1] << 1u8; source[2] << 2u8; sha256(source);`,
			func(value object.Object) bool {
				text, ok := value.(*object.String)
				return ok && text.Value == "ae4b3280e56e2faf83f414a6e3dabe9d5fbe18976544c05fed121accb85b53fc"
			},
		},
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
		if value := machine.LastPoppedStackElem(); !test.check(value) {
			t.Fatalf("unexpected value %T %s", value, value.Inspect())
		}
	}
}

func TestVMBinaryFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roundtrip.bin")
	program := parse(fmt.Sprintf(`
        var data << bytes(4);
        writeU32BE(data, 0, 0x4E45531Au32);
        writeBytes(%q, data);
        var loaded << readBytes(%q);
        bytesEqual(data, loaded);
    `, path, path))
	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}
	machine := New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("VM error: %v", err)
	}
	value, ok := machine.LastPoppedStackElem().(*object.Boolean)
	if !ok || !value.Value {
		t.Fatalf("unexpected round-trip result: %T %s", machine.LastPoppedStackElem(), machine.LastPoppedStackElem().Inspect())
	}
}
