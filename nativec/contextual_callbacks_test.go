package nativec_test

import (
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestContextualCallbackBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/contextual_callbacks.zum")
	if output != "42\n" {
		t.Fatalf("unexpected contextual callback output %q", output)
	}
}

func TestContextualCallbackMIRReachesNativeGeneratorWithoutUnknownSignature(t *testing.T) {
	result, diagnostics := pipeline.Build("callbacks.zum", `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var double << fct(value) { value * 2i32; };
        unsafe { apply(21i32, double); }
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	if strings.Contains(result.DumpMIR(), "function double(value) -> unknown") {
		t.Fatalf("native backend received an unknown callback signature:\n%s", result.DumpMIR())
	}
	_, nativeDiagnostics := nativec.Generate(result.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
}
