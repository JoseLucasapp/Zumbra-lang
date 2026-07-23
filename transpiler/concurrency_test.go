package transpiler

import (
	"strings"
	"testing"

	"zumbra/pipeline"
)

func TestLegacyGoTranspilerRejectsZ9ConcurrencyClearly(t *testing.T) {
	result, diagnostics := pipeline.Build("concurrency.zum", `
        fct answer() { 42; }
        var task << spawn answer();
        await task;
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	_, err := ZumbraTranspilerPipeline(result)
	if err == nil || !strings.Contains(err.Error(), "VM and native backend") {
		t.Fatalf("expected explicit backend guidance, got %v", err)
	}
}
