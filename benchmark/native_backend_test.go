package main

import (
	"testing"
	"zumbra/nativec"
	"zumbra/pipeline"
)

func BenchmarkNativeCGeneration(b *testing.B) {
	result, diagnostics := pipeline.Build("benchmark.zum", irBenchmarkSource, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		b.Fatalf("pipeline failed: %v", diagnostics)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, diagnostics := nativec.Generate(result.MIR); len(diagnostics) != 0 {
			b.Fatalf("native generation failed: %v", diagnostics)
		}
	}
}
