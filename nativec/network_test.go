package nativec_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ10TCPUDPAndDNSBuildAndRunNatively(t *testing.T) {
	output := buildAndRunSource(t, `
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
        show(sizeOf(response));
        streamClose(client);
        await server;
        listenerClose(listener);

        var receiver << udpBind("127.0.0.1", 0);
        var sender << udpBind("127.0.0.1", 0);
        udpSendTo(sender, "127.0.0.1", udpPort(receiver), "ok");
        var packet << udpReceiveFromTimeout(receiver, 16, 1000);
        show(packet[3]);
        show(sizeOf(packet[0]));
        udpClose(sender);
        udpClose(receiver);
        show(sizeOf(dnsLookup("localhost")) > 0);
    `)
	if output != "4\ntrue\n2\ntrue\n" {
		t.Fatalf("unexpected native network output %q", output)
	}
}

func TestZ10TLSBuildAndRunNatively(t *testing.T) {
	certificate, err := filepath.Abs(filepath.Join("..", "code_examples", "native", "z10_test_cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := filepath.Abs(filepath.Join("..", "code_examples", "native", "z10_test_key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	source := fmt.Sprintf(`
        fct echo(listener) {
            var connection << listenerAccept(listener);
            var payload << streamReadExact(connection, 4);
            streamWriteAll(connection, payload);
            streamClose(connection);
            return;
        }
        var listener << tlsListen("127.0.0.1", 0, %q, %q);
        var server << spawn echo(listener);
        var client << tlsConnectTimeout("127.0.0.1", listenerPort(listener), "localhost", true, 1000);
        streamWriteAll(client, "tls!");
        show(sizeOf(streamReadExact(client, 4)));
        streamClose(client);
        await server;
        listenerClose(listener);
    `, certificate, key)
	output := buildAndRunSource(t, source)
	if output != "4\n" {
		t.Fatalf("unexpected native TLS output %q", output)
	}
}

func TestZ10NativeRuntimeIsConditionallyLinked(t *testing.T) {
	plain, diagnostics := pipeline.Build("plain.zum", `show(42);`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatal(pipeline.FormatDiagnostics(diagnostics))
	}
	plainSources, nativeDiagnostics := nativec.Generate(plain.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	if strings.Contains(string(plainSources.Runtime), "#define ZUMBRA_ENABLE_NETWORK") {
		t.Fatal("sequential program unexpectedly enabled network runtime")
	}

	network, diagnostics := pipeline.Build("network.zum", `var listener << tcpListen("127.0.0.1", 0); listenerClose(listener);`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatal(pipeline.FormatDiagnostics(diagnostics))
	}
	networkSources, nativeDiagnostics := nativec.Generate(network.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	if !strings.Contains(string(networkSources.Runtime), "#define ZUMBRA_ENABLE_NETWORK") {
		t.Fatal("network program did not enable network runtime")
	}
	if strings.Contains(string(networkSources.Runtime), "#define ZUMBRA_ENABLE_TLS") {
		t.Fatal("plain TCP program unexpectedly enabled TLS")
	}
}
