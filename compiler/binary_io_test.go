package compiler

import "testing"

func TestCompileBinaryBuiltins(t *testing.T) {
	program := parse(`
        var data << bytes(16);
        writeU64BE(data, 0, 0x0102030405060708u64);
        var value << readU64BE(data, 0);
        var target << bytes(16);
        copyBytes(target, 0, data, 0, 8);
        bytesEqual(target, data);
        sha256(data);
    `)
	compiled := New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %v", err)
	}
}
