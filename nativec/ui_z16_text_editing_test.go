package nativec

import (
	"strings"
	"testing"
)

func TestZ16TextEditingRuntimeIsEmitted(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	runtime := string(data)
	for _, token := range []string{"z_ui_text_handle_key", "selectionStart", "ctrl+v", "z_ui_text_caret_from_x", "scrollbarThumb", "scrollbarAvoidContent"} {
		if !strings.Contains(runtime, token) {
			t.Fatalf("native UI runtime missing %s", token)
		}
	}
}
