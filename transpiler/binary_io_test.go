package transpiler

import (
	"strings"
	"testing"
)

func TestTranspilerTranslatesBinaryOperations(t *testing.T) {
	got, err := ZumbraTranspiler(`
        var data << readBytes("game.nes");
        writeU16LE(data, 0, 0x1234u16);
        var value << readU16LE(data, 0);
        var target << bytes(16);
        copyBytes(target, 0, data, 0, 16);
        var same << bytesEqual(target, data);
        var hash << sha256(data);
        writeBytes("copy.bin", target);
    `)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`var data = zReadBytes("game.nes")`,
		`zWriteU16LE(data, 0, zU16(0x1234))`,
		`var value = zReadU16LE(data, 0)`,
		`zCopyBytes(target, 0, data, 0, 16)`,
		`var same = zBytesEqual(target, data)`,
		`var hash = zSHA256(data)`,
		`zWriteBytes("copy.bin", target)`,
		`"encoding/binary"`,
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q\n%s", expected, got)
		}
	}
}
