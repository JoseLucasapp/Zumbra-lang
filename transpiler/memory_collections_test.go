package transpiler

import (
	"strings"
	"testing"
)

func TestTranspilerTranslatesMemoryCollections(t *testing.T) {
	got, err := ZumbraTranspiler(`
        var memory << bytes(8);
        memory[1] << 0xA9u8;
        var value << memory[1];
        var regs << arrayOf("u16", 4);
        var view << slice(regs, 1, 3);
        fill(view, 0u16);
    `)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"var memory = zBytes(8)",
		"zSet(memory, 1, zU8(0xA9))",
		"var value = zGet(memory, 1)",
		`var regs = zArrayOf("u16", 4)`,
		"var view = zSlice(regs, 1, 3)",
		"zFill(view, zU16(0))",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("missing %q\n%s", expected, got)
		}
	}
}
