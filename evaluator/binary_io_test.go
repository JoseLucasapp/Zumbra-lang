package evaluator

import (
	"fmt"
	"path/filepath"
	"testing"
	"zumbra/object"
)

func TestEvaluateBinaryEndianCopyAndHash(t *testing.T) {
	result := testEval(`
        var source << bytes(8);
        writeU16LE(source, 0, 0x1234u16);
        writeU16BE(source, 2, 0xABCDu16);
        var target << bytes(4);
        copyBytes(target, 0, source, 0, 4);
        var same << bytesEqual(target, slice(source, 0, 4));
        if (same) {
            readU16LE(target, 0) + readU16BE(target, 2);
        } else {
            0u16;
        }
    `)
	value, ok := result.(*object.FixedInteger)
	if !ok || value.Kind != object.FixedU16 || value.UnsignedValue() != uint64(uint16(0x1234+0xABCD)) {
		t.Fatalf("unexpected result: %T %v", result, result)
	}
}

func TestEvaluateSHA256(t *testing.T) {
	result := testEval(`
        var data << bytes(3);
        data[1] << 1u8;
        data[2] << 2u8;
        sha256(data);
    `)
	value, ok := result.(*object.String)
	if !ok || value.Value != "ae4b3280e56e2faf83f414a6e3dabe9d5fbe18976544c05fed121accb85b53fc" {
		t.Fatalf("unexpected hash: %T %v", result, result)
	}
}

func TestEvaluateBinaryFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "roundtrip.bin")
	result := testEval(fmt.Sprintf(`
        var data << bytes(4);
        writeU32BE(data, 0, 0x4E45531Au32);
        writeBytes(%q, data);
        var loaded << readBytes(%q);
        bytesEqual(data, loaded);
    `, path, path))
	value, ok := result.(*object.Boolean)
	if !ok || !value.Value {
		t.Fatalf("unexpected round-trip result: %T %v", result, result)
	}
}
