package nativec_test

import (
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ16FinalSelectDropdownRunsHeadlessNatively(t *testing.T) {
	expected := "true\nPrimeira\nSegunda\nfalse\n"
	if output := buildAndRunZ8(t, "code_examples/core/ui_select_dropdown.zum"); output != expected {
		t.Fatalf("output=%q", output)
	}
}

func TestZ16FinalNativeRuntimeContainsDropdownBlurAndImagePreviewSupport(t *testing.T) {
	result, diagnostics := pipeline.Build("ui-final.zum", `var app << desktopApp({"backend":"headless"}); var w << app.window({"title":"UI"}); var s << uiSelect({"options":["A","B"],"value":"A"}, []); var image << uiImage({"path":"cover.png","fit":"contain"}, []); var modal << uiModal({"visible":true,"width":500,"height":400,"backdropBlur":6,"overflow":"hidden"}, [s,image]); uiMount(app,w,modal,{});`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native: %#v", nativeDiagnostics)
	}
	runtime := string(sources.Runtime)
	for _, expected := range []string{
		"z_ui_render_select_popup", "popupOffset", "selectedOptionBackground",
		"z_ui_blur_backdrop", "SDL_RenderReadPixels", "SDL_SetRenderDrawBlendMode",
		"IMG_Load", "gdk_pixbuf_new_from_file", "z_ui_render_image_file",
		"z_ui_clips_children",
	} {
		if !strings.Contains(runtime, expected) {
			t.Fatalf("missing %q", expected)
		}
	}
}
