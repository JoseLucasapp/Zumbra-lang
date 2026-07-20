package runtime

import (
	"strings"
	"testing"
)

func TestRuntimeContainsMemoryCollectionSupport(t *testing.T) {
	source := Runtime()
	for _, name := range []string{"func zBytes", "func zArrayOf", "func zGet", "func zSet", "func zSlice", "func zFill"} {
		if !strings.Contains(source, name) {
			t.Fatalf("runtime missing %s", name)
		}
	}
}
