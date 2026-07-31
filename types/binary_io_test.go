package types

import "testing"

func TestBinaryBuiltinsAreTyped(t *testing.T) {
	errors := checkInput(t, `
        var memory << bytes(16);
        writeU16LE(memory, 0, 0x1234u16);
        var a << readU16LE(memory, 0);
        var b << readU32BE(memory, 4);
        var target << bytes(16);
        copyBytes(target, 0, memory, 0, 16);
        var same << bytesEqual(memory, target);
        var hash << sha256(memory);
    `)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestBinaryBuiltinsRejectNonByteBuffers(t *testing.T) {
	errors := checkInput(t, `
        var words << arrayOf("u16", 4);
        readU16LE(words, 0);
    `)
	if len(errors) == 0 {
		t.Fatal("expected byte-buffer type error")
	}
}
