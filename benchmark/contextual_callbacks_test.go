package main

import (
	"testing"

	"zumbra/pipeline"
)

func BenchmarkContextualCallbackPipeline(b *testing.B) {
	const source = `
        extern "C" { fct apply(value: i32, cb: callback(i32) -> i32) -> i32; }
        var double << fct(value) { value * 2i32; };
        unsafe { apply(21i32, double); }
    `
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, diagnostics := pipeline.Build("contextual-callback-benchmark.zum", source, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.Types == nil || result.HIR == nil || result.MIR == nil {
			b.Fatalf("contextual callback pipeline failed: %v", diagnostics)
		}
	}
}
