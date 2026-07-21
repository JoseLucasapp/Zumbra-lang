package runtime

import (
	"strings"
	"testing"
)

func TestRuntimeContainsBinaryIOSupport(t *testing.T) {
	source := Runtime()
	for _, name := range []string{
		"func zReadBytes", "func zWriteBytes",
		"func zReadU16LE", "func zReadU32BE", "func zReadU64LE",
		"func zWriteU16LE", "func zWriteU32BE", "func zWriteU64LE",
		"func zCopyBytes", "func zBytesEqual", "func zSHA256",
	} {
		if !strings.Contains(source, name) {
			t.Fatalf("runtime missing %s", name)
		}
	}
}
