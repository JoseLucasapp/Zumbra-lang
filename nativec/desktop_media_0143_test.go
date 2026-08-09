package nativec_test

import (
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZumbra0143DesktopMediaBuildsNatively(t *testing.T) {
	source := `
var app << desktopApp({"backend":"headless","name":"Media"});
var window << desktopWindow(app,{"title":"Media","width":2,"height":2});
var pixels << bytes(16);
show(desktopWindowPresentRGBA(window,pixels,2,2));
desktopWindowSetVSync(window,true);
show(desktopKeyDown(app,29));
show(desktopGamepadButton(app,1,0));
var samples << bytes(4);
fill(samples,32u8);
show(desktopAudioQueue(app,samples,80,false));
show(desktopAudioQueued(app));
show(sizeOf(processArgs()) > 0);
show(unixTimeSeconds() > 0u64);
show(createFile("data/zumbra-runtime.txt","0.14.5"));
desktopClose(app);
`
	result, diagnostics := pipeline.Build("desktop-media.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	sources, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	runtime := string(sources.Runtime)
	for _, expected := range []string{
		"desktopWindowPresentRGBA",
		"desktopAudioQueue",
		"desktopGamepadButton",
		"z_runtime_set_args",
		"createFile",
	} {
		if !strings.Contains(runtime, expected) && !strings.Contains(string(sources.Program), expected) {
			t.Fatalf("generated native sources do not contain %q", expected)
		}
	}
}
