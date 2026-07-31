package main

import (
	"testing"
	"zumbra/pipeline"
)

const irBenchmarkSource = `
struct Point { x: int; y: int; fct move(dx, dy) { self.x << self.x + dx; self.y << self.y + dy; } }
var point << Point(10, 20);
point.move(2, 3);
var folded << 2 + 3 * 4;
point.x + point.y + folded;
`

func BenchmarkTypedPipeline(b *testing.B) {
	for i := 0; i < b.N; i++ {
		result, diagnostics := pipeline.Build("benchmark.zum", irBenchmarkSource, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.MIR == nil {
			b.Fatalf("pipeline failed: %v", diagnostics)
		}
	}
}

func BenchmarkTypedPipelineWithoutOptimization(b *testing.B) {
	for i := 0; i < b.N; i++ {
		result, diagnostics := pipeline.Build("benchmark.zum", irBenchmarkSource, pipeline.Options{Optimize: false})
		if len(diagnostics) != 0 || result.MIR == nil {
			b.Fatalf("pipeline failed: %v", diagnostics)
		}
	}
}
