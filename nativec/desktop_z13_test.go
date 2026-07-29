package nativec_test

import (
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ13DesktopExampleBuildsAndRunsHeadlessNatively(t *testing.T) {
	expected := "headless\nZumbra Desktop 0.10.3\n960\n640\nhello desktop\ntrue\n7\n"
	if output := buildAndRunZ8(t, "code_examples/core/desktop_runtime.zum"); output != expected {
		t.Fatalf("unexpected native output %q", output)
	}
}

func TestZ13DesktopRuntimeIsConditionallyEnabled(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		enabled bool
	}{
		{"plain", `show(42);`, false},
		{"desktop", `var app << desktopApp({"backend": "headless"}); desktopClose(app);`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, diagnostics := pipeline.Build(test.name+".zum", test.source, pipeline.Options{Optimize: true})
			if len(diagnostics) != 0 {
				t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
			}
			sources, nativeDiagnostics := nativec.Generate(result.MIR)
			if len(nativeDiagnostics) != 0 {
				t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
			}
			runtimeSource := string(sources.Runtime)
			enabled := strings.Contains(runtimeSource, "#define ZUMBRA_ENABLE_DESKTOP 1")
			if enabled != test.enabled {
				t.Fatalf("desktop runtime enabled=%t, expected %t", enabled, test.enabled)
			}
			if test.enabled {
				for _, expected := range []string{"z_desktop_app_new", "SDL_CreateWindow", "desktopProcessWait"} {
					if !strings.Contains(runtimeSource, expected) {
						t.Fatalf("missing %q from generated desktop runtime", expected)
					}
				}
			}
		})
	}
}
