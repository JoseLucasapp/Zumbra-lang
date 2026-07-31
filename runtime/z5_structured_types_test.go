package runtime

import (
	"strings"
	"testing"
)

func TestRuntimeContainsZ5Support(t *testing.T) {
	source := Runtime()
	for _, expected := range []string{"type zStructDefinition", "func zConstruct", "func zGetAttr", "func zSetAttr", "func zCallMethod", "func zEnum", "func zMatch", "func zIntegerTarget"} {
		if !strings.Contains(source, expected) {
			t.Fatalf("runtime missing %s", expected)
		}
	}
}
