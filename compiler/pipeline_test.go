package compiler

import (
	"testing"
	"zumbra/pipeline"
)

func TestCompilePipelineAcceptsAnalyzedResult(t *testing.T) {
	result, diagnostics := pipeline.Build("test.zum", `var value << 2 + 3 * 4; value;`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", diagnostics)
	}
	compiled := New()
	if err := compiled.CompilePipeline(result); err != nil {
		t.Fatal(err)
	}
	if len(compiled.Bytecode().Instructions) == 0 {
		t.Fatal("expected bytecode")
	}
}
