package semantic

import "testing"

func TestResolveBinaryIOBuiltins(t *testing.T) {
	errors := resolveInput(`
        var data << bytes(16);
        writeU16LE(data, 0, 0x1234u16);
        var value << readU16LE(data, 0);
        var target << bytes(16);
        copyBytes(target, 0, data, 0, 16);
        var same << bytesEqual(data, target);
        var hash << sha256(data);
        writeBytes("tmp/data.bin", target);
        var loaded << readBytes("tmp/data.bin");
    `)
	if len(errors) != 0 {
		t.Fatalf("expected no resolver errors, got %v", errors)
	}
}
