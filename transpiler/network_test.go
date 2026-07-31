package transpiler

import (
	"strings"
	"testing"

	"zumbra/pipeline"
)

func TestLegacyGoTranspilerRejectsZ10NetworkingClearly(t *testing.T) {
	result, diagnostics := pipeline.Build("network.zum", `
        var listener << tcpListen("127.0.0.1", 0);
        listenerClose(listener);
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	_, err := ZumbraTranspilerPipeline(result)
	if err == nil || !strings.Contains(err.Error(), "Z10 networking") {
		t.Fatalf("expected explicit networking diagnostic, got %v", err)
	}
}
