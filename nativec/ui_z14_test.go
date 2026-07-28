package nativec_test

import (
	"strings"
	"testing"
	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ14GUIExampleBuildsAndRunsHeadlessNatively(t *testing.T) {
	expected := "headless\nZumbra UI\ntrue\nclicked\n640\ntrue\ntrue\n"
	if output := buildAndRunZ8(t, "code_examples/core/gui_toolkit.zum"); output != expected {
		t.Fatalf("output=%q", output)
	}
}
func TestZ14UIRuntimeIsConditionallyEnabled(t *testing.T) {
	result, diagnostics := pipeline.Build("ui.zum", `var app << desktopApp({"backend":"headless"}); var w << app.window({"title":"UI"}); var n << uiText({"text":"x"}, []); uiMount(app,w,n,{});`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native: %#v", nativeDiagnostics)
	}
	runtime := string(sources.Runtime)
	for _, expected := range []string{"#define ZUMBRA_ENABLE_UI 1", "z_ui_render_context", "TTF_OpenFont", "TTF_RenderText_Blended", "fontFamily", "uiCanvasCommand"} {
		if !strings.Contains(runtime, expected) {
			t.Fatalf("missing %q", expected)
		}
	}
}

func TestZ14PointOneNativeRuntimeContainsThemeAndUTF8FontSupport(t *testing.T) {
	result, diagnostics := pipeline.Build("ui-font.zum", `var app << desktopApp({"backend":"headless"}); var w << app.window({"title":"UI"}); var n << uiText({"text":"ação e café","fontSize":22,"fontWeight":"bold"}, []); var c << uiMount(app,w,n,{"theme":uiTheme("dark")}); uiSnapshot(c);`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native: %#v", nativeDiagnostics)
	}
	runtime := string(sources.Runtime)
	for _, expected := range []string{"libSDL3_ttf.so.0", "TTF_GetStringSize", "TTF_RenderText_Blended", "#141822", "lineHeight", "ZUMBRA_UI_FONT"} {
		if !strings.Contains(runtime, expected) {
			t.Fatalf("missing %q", expected)
		}
	}
}
