package binarydata

import (
	"os"
	"path/filepath"
	"testing"

	"zumbra/object"
)

func TestEndianReadWriteAndHash(t *testing.T) {
	memory := &object.ByteArray{Data: make([]byte, 16)}

	if _, err := WriteUnsigned(memory, &object.Integer{Value: 0}, object.NewFixedIntegerRaw(object.FixedU16, 0x1234), 2, LittleEndian); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteUnsigned(memory, &object.Integer{Value: 2}, object.NewFixedIntegerRaw(object.FixedU16, 0xABCD), 2, BigEndian); err != nil {
		t.Fatal(err)
	}

	le, err := ReadUnsigned(memory, &object.Integer{Value: 0}, 2, LittleEndian)
	if err != nil {
		t.Fatal(err)
	}
	be, err := ReadUnsigned(memory, &object.Integer{Value: 2}, 2, BigEndian)
	if err != nil {
		t.Fatal(err)
	}
	if le.(*object.FixedInteger).UnsignedValue() != 0x1234 || be.(*object.FixedInteger).UnsignedValue() != 0xABCD {
		t.Fatalf("unexpected endian values: %s %s", le.Inspect(), be.Inspect())
	}

	hash, err := SHA256(&object.ByteArray{Data: []byte{0, 1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	if hash.(*object.String).Value != "ae4b3280e56e2faf83f414a6e3dabe9d5fbe18976544c05fed121accb85b53fc" {
		t.Fatalf("unexpected hash %s", hash.Inspect())
	}
}

func TestCopySupportsOverlappingRanges(t *testing.T) {
	memory := &object.ByteArray{Data: []byte{1, 2, 3, 4, 5}}
	_, err := Copy(memory, &object.Integer{Value: 1}, memory, &object.Integer{Value: 0}, &object.Integer{Value: 4})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 1, 2, 3, 4}
	for i := range want {
		if memory.Data[i] != want[i] {
			t.Fatalf("copy mismatch at %d: got %v want %v", i, memory.Data, want)
		}
	}
}

func TestReadAndWriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.bin")
	written, err := WriteFile(&object.String{Value: path}, &object.ByteArray{Data: []byte{0x4E, 0x45, 0x53, 0x1A}})
	if err != nil {
		t.Fatal(err)
	}
	if written.(*object.Integer).Value != 4 {
		t.Fatalf("expected 4 bytes, got %s", written.Inspect())
	}

	loaded, err := ReadFile(&object.String{Value: path})
	if err != nil {
		t.Fatal(err)
	}
	if string(loaded.(*object.ByteArray).Data) != string([]byte{0x4E, 0x45, 0x53, 0x1A}) {
		t.Fatalf("unexpected file bytes %v", loaded.(*object.ByteArray).Data)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsInvalidRangesAndNonByteArrays(t *testing.T) {
	memory := &object.ByteArray{Data: make([]byte, 2)}
	if _, err := ReadUnsigned(memory, &object.Integer{Value: 1}, 2, LittleEndian); err == nil {
		t.Fatal("expected out-of-range error")
	}

	typed := &object.TypedArray{Kind: object.FixedU16, Data: make([]byte, 4), Length: 2}
	if _, err := Bytes(typed); err == nil {
		t.Fatal("expected non-byte typed array error")
	}
}
