package transpiler

import (
	"strings"
	"testing"
	"zumbra/pipeline"
)

func TestTranspilerPipelineRequiresAnalyzedSource(t *testing.T) {
	result, diagnostics := pipeline.Build("test.zum", `var value << 2 + 3; show(value);`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", diagnostics)
	}
	generated, err := ZumbraTranspilerPipeline(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(generated, "package main") {
		t.Fatalf("unexpected Go output: %s", generated)
	}
}
