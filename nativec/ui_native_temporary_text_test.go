package nativec

import (
	"strings"
	"testing"
)

func TestNativeUIFittedTextUsesMatchingAllocator(t *testing.T) {
	data, err := runtimeFiles.ReadFile("runtime/zumbra_ui.inc")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	start := strings.Index(source, "static char*z_ui_fit_text_native")
	if start < 0 {
		t.Fatal("native fitted-text helper was not found")
	}
	end := strings.Index(source[start:], "static void z_ui_text_aligned")
	if end < 0 {
		t.Fatal("native aligned-text helper was not found")
	}
	fitSource := source[start : start+end]
	if strings.Contains(fitSource, "z_strdup(") || strings.Contains(fitSource, "z_alloc(") {
		t.Fatal("fitted text returned to free() must not use the runtime allocation registry")
	}
	for _, token := range []string{
		"static char*z_ui_temporary_string",
		"char*copy=(char*)malloc(length+1)",
		"char*fitted=z_ui_fit_text_native",
		"if(fitted==NULL)return",
		"free(fitted)",
	} {
		if !strings.Contains(source, token) {
			t.Fatalf("native fitted-text ownership fix missing %q", token)
		}
	}
}
