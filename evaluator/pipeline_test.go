package evaluator

import (
	"testing"
	"zumbra/object"
	"zumbra/pipeline"
)

func TestEvalPipelineUsesCanonicalResult(t *testing.T) {
	result, diagnostics := pipeline.Build("test.zum", `var value << 7; value + 5;`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", diagnostics)
	}
	value := EvalPipeline(result, object.NewEnvironment())
	integer, ok := value.(*object.Integer)
	if !ok || integer.Value != 12 {
		t.Fatalf("expected 12, got %T %v", value, value)
	}
}
