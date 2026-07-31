package main

import (
	"testing"
	"zumbra/object/builtins"
	"zumbra/pipeline"
)

func BenchmarkHTTPPipeline(b *testing.B) {
	const source = `
        var app << httpApp();
        app.get("/health", fct(request, response) {
            httpJson(200, {"status":"ok", "runtime":"zumbra"});
        });
        var server << app.listen("127.0.0.1", 0);
        httpShutdown(server, 1000);
    `
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		result, diagnostics := pipeline.Build("http-benchmark.zum", source, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.MIR == nil {
			b.Fatalf("pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
		}
	}
}

func BenchmarkJSONRoundTrip(b *testing.B) {
	source := builtins.NewString(`{"name":"zumbra","value":42,"active":true}`)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		parsed := builtins.JSONParseZ11Builtin().Fn(source)
		encoded := builtins.JSONStringifyBuiltin().Fn(parsed)
		if encoded.Type() == "ERROR" {
			b.Fatal(encoded.Inspect())
		}
	}
}
