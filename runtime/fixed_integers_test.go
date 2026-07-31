package runtime

import (
	"strings"
	"testing"
)

func TestRuntimeContainsFixedIntegerSupport(t *testing.T) {
	runtime := Runtime()
	for _, expected := range []string{
		"func zU8",
		"func zU64",
		"func zI8",
		"func wrapAdd",
		"func checkedAdd",
		"func satAdd",
	} {
		if !strings.Contains(runtime, expected) {
			t.Fatalf("runtime missing %q", expected)
		}
	}
}
