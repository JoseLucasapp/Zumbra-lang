package pipeline

import (
	"strings"
	"testing"
)

func TestContextualCallbacksAreConcreteAcrossPipeline(t *testing.T) {
	result, diagnostics := Build("callbacks.zum", `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var double << fct(value) { value * 2i32; };
        unsafe { apply(21i32, double); }
    `, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", FormatDiagnostics(diagnostics))
	}
	for label, dump := range map[string]string{
		"HIR": result.DumpHIR(),
		"MIR": result.DumpMIR(),
	} {
		if strings.Contains(dump, `function double(value) -> unknown`) || strings.Contains(dump, `fct(unknown) -> unknown`) {
			t.Fatalf("%s retained an unknown callback signature:\n%s", label, dump)
		}
		if !strings.Contains(dump, "fct(i32) -> i32") {
			t.Fatalf("%s does not contain the contextual callback type:\n%s", label, dump)
		}
	}
}
