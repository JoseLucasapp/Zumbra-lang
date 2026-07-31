package main

import (
	"os"
	"path/filepath"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func BenchmarkZ8ModulePipeline(b *testing.B) {
	dir := b.TempDir()
	dependency := filepath.Join(dir, "math.zum")
	entry := filepath.Join(dir, "app.zum")
	if err := os.WriteFile(dependency, []byte(`pub const BASE << 40; pub fct add(a, b) { a + b; }`), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte(`import "math.zum" as math; math.add(math.BASE, 2);`), 0o644); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, diagnostics := pipeline.BuildFile(entry, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.Modules == nil || len(result.Modules.Units) != 2 {
			b.Fatalf("module pipeline failed: %v", diagnostics)
		}
	}
}

func BenchmarkZ8FFIAdapterGeneration(b *testing.B) {
	result, diagnostics := pipeline.Build("ffi-benchmark.zum", `
        extern "C" {
            fct add(left: i32, right: i32) -> i32;
            fct apply(value: i32, cb: callback(i32) -> i32) -> i32;
        }
        var double << fct(value) { value * 2i32; };
        unsafe { add(20i32, 22i32); apply(21i32, double); }
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		b.Fatalf("pipeline failed: %v", diagnostics)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, nativeDiagnostics := nativec.Generate(result.MIR); len(nativeDiagnostics) != 0 {
			b.Fatalf("FFI generation failed: %v", nativeDiagnostics)
		}
	}
}
