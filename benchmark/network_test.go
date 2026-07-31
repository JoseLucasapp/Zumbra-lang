package main

import (
	"net"
	"testing"

	"zumbra/object"
	"zumbra/object/builtins"
	"zumbra/pipeline"
)

func BenchmarkNetworkPipeline(b *testing.B) {
	const source = `
        fct echo(listener) {
            var connection << listenerAccept(listener);
            var payload << streamReadExact(connection, 4);
            streamWriteAll(connection, payload);
            streamClose(connection);
            return;
        }
        var listener << tcpListen("127.0.0.1", 0);
        var server << spawn echo(listener);
        var client << tcpConnectTimeout("127.0.0.1", listenerPort(listener), 1000);
        streamWriteAll(client, "ping");
        var response << streamReadExact(client, 4);
        streamClose(client);
        await server;
        listenerClose(listener);
    `
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		result, diagnostics := pipeline.Build("network-benchmark.zum", source, pipeline.Options{Optimize: true})
		if len(diagnostics) != 0 || result.MIR == nil {
			b.Fatalf("pipeline failed: %s", pipeline.FormatDiagnostics(diagnostics))
		}
	}
}

func BenchmarkStreamRoundTrip(b *testing.B) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()
	client := &object.NetStream{Conn: left}
	server := &object.NetStream{Conn: right}
	payload := builtins.NewString("ping")
	buffer := make(chan object.Object, 1)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		go func() {
			buffer <- builtins.StreamReadBuiltin(true, false).Fn(server, builtins.NewInteger(4))
		}()
		written := builtins.StreamWriteBuiltin(true).Fn(client, payload)
		if err, ok := written.(*object.Error); ok {
			b.Fatal(err.Message)
		}
		result := <-buffer
		if err, ok := result.(*object.Error); ok {
			b.Fatal(err.Message)
		}
	}
}
