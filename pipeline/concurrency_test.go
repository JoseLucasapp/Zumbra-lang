package pipeline

import (
	"strings"
	"testing"
)

func TestConcurrencyIsTypedAcrossHIRAndMIR(t *testing.T) {
	result, diagnostics := Build("concurrency.zum", `
        fct answer() { 42; }
        var task << spawn answer();
        var result << await task;
        result;
    `, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", FormatDiagnostics(diagnostics))
	}
	hirDump := result.DumpHIR()
	mirDump := result.DumpMIR()
	for _, expected := range []string{"task<int>", "spawn", "await"} {
		if !strings.Contains(strings.ToLower(hirDump), expected) && !strings.Contains(strings.ToLower(mirDump), expected) {
			t.Fatalf("pipeline dump does not contain %q:\nHIR:\n%s\nMIR:\n%s", expected, hirDump, mirDump)
		}
	}
	if strings.Contains(hirDump, `var name="task" : unknown`) || strings.Contains(mirDump, `declare "task"`) && strings.Contains(mirDump, `: unknown`) {
		t.Fatalf("task type was not preserved:\nHIR:\n%s\nMIR:\n%s", hirDump, mirDump)
	}
}
