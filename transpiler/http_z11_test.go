package transpiler

import (
	"strings"
	"testing"
	"zumbra/pipeline"
)

func TestLegacyGoTranspilerRejectsZ11HTTPClearly(t *testing.T) {
	result, diagnostics := pipeline.Build("http.zum", `var app << httpApp();`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	_, err := ZumbraTranspilerPipeline(result)
	if err == nil || !strings.Contains(err.Error(), "Z11 HTTP") {
		t.Fatalf("expected explicit HTTP diagnostic, got %v", err)
	}
}
