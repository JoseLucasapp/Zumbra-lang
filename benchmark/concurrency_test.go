package main

import (
	"testing"

	"zumbra/object"
	"zumbra/pipeline"
)

const concurrencyBenchmarkSource = `
    fct square(value) { value * value; }
    var task << spawn square(21);
    var result << await task;
    var counter << atomicInt(result);
    atomicAdd(counter, 1);
`

func BenchmarkConcurrencyPipeline(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, diagnostics := pipeline.Build("concurrency-benchmark.zum", concurrencyBenchmarkSource, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.HIR == nil || result.MIR == nil {
			b.Fatalf("concurrency pipeline failed: %v", diagnostics)
		}
	}
}

func BenchmarkChannelRoundTrip(b *testing.B) {
	channel := object.NewChannel(1)
	value := &object.Integer{Value: 42}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := channel.Send(value); err != nil {
			b.Fatal(err)
		}
		received, open := channel.Receive()
		if !open || received != value {
			b.Fatal("channel round trip failed")
		}
	}
}
