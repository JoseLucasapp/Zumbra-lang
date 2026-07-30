package nativec

import (
	"os"
	"strings"
	"testing"
)

func TestZ16NativeSidebarAndLineLabelsArePresent(t *testing.T) {
	data, err := os.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, required := range []string{
		`strcmp(n->kind,"menu")!=0`,
		`z_ui_value_text(labels.as.array->items[i])`,
		`showValues`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("native UI runtime missing %q", required)
		}
	}
}
