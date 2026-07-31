package main

import (
	"testing"

	"zumbra/pipeline"
)

const callInferenceBenchmarkSource = `
    fct square(value) { value * value; }
    fct produce(messages) { send(messages, 7); return; }
    var task << spawn square(21);
    var messages << channel(1);
    var producer << spawn produce(messages);
    var value << receive(messages);
    await producer;
    var result << await task;
`

func BenchmarkCallInferencePipeline(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, diagnostics := pipeline.Build("call-inference-benchmark.zum", callInferenceBenchmarkSource, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.Types == nil || result.HIR == nil || result.MIR == nil {
			b.Fatalf("call inference pipeline failed: %v", diagnostics)
		}
		square, ok := result.Types.Global("square")
		if !ok || square.String() != "fct(int) -> int" {
			b.Fatalf("call inference did not specialize square: %v", square)
		}
	}
}
