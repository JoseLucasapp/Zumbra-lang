package transpiler

import (
	"strings"
	"testing"
)

func TestTranspilerTranslatesFixedIntegerSyntax(t *testing.T) {
	input := `
		var opcode << 0xFFu8;
		var address << u16(0x8000);
		var mask << opcode band 0x0Fu8;
		var shifted << mask shl 1;
		var saturated << satAdd(opcode, 1u8);
	`

	got, err := ZumbraTranspiler(input)
	if err != nil {
		t.Fatalf("transpiler error: %v", err)
	}

	for _, expected := range []string{
		"var opcode = zU8(0xFF)",
		"var address = zU16(0x8000)",
		"opcode & zU8(0x0F)",
		"mask << 1",
		"satAdd(opcode, zU8(1))",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("transpiled output missing %q\n%s", expected, got)
		}
	}
}
