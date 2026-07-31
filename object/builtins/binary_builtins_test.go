package builtins

import (
	"path/filepath"
	"testing"

	"zumbra/object"
)

func TestBinaryBuiltins(t *testing.T) {
	memory := &object.ByteArray{Data: make([]byte, 8)}
	result := WriteUnsignedBuiltin(4, "little", "writeU32LE").Fn(memory, &object.Integer{Value: 0}, object.NewFixedIntegerRaw(object.FixedU32, 0x78563412))
	if result.Type() == object.ERROR_OBJ {
		t.Fatal(result.Inspect())
	}
	read := ReadUnsignedBuiltin(4, "little", "readU32LE").Fn(memory, &object.Integer{Value: 0})
	value, ok := read.(*object.FixedInteger)
	if !ok || value.Kind != object.FixedU32 || value.UnsignedValue() != 0x78563412 {
		t.Fatalf("unexpected read result %T %s", read, read.Inspect())
	}

	hash := SHA256Builtin().Fn(&object.ByteArray{Data: []byte{0, 1, 2}})
	if hash.(*object.String).Value != "ae4b3280e56e2faf83f414a6e3dabe9d5fbe18976544c05fed121accb85b53fc" {
		t.Fatalf("unexpected hash %s", hash.Inspect())
	}
}

func TestBinaryFileBuiltins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rom.bin")
	memory := &object.ByteArray{Data: []byte{1, 2, 3}}
	written := WriteBytesBuiltin().Fn(&object.String{Value: path}, memory)
	if written.Type() == object.ERROR_OBJ {
		t.Fatal(written.Inspect())
	}
	loaded := ReadBytesBuiltin().Fn(&object.String{Value: path})
	if loaded.Type() == object.ERROR_OBJ {
		t.Fatal(loaded.Inspect())
	}
	equal := BytesEqualBuiltin().Fn(memory, loaded)
	if !equal.(*object.Boolean).Value {
		t.Fatal("expected buffers to be equal")
	}
}
