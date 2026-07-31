package pipeline

import (
	"strings"
	"testing"
)

func TestNetworkTypesFlowThroughHIRAndMIR(t *testing.T) {
	result, diagnostics := Build("network-types.zum", `
        var listener << tcpListen("127.0.0.1", 0);
        var connection << tcpConnectTimeout("127.0.0.1", listenerPort(listener), 1000);
        var payload << streamRead(connection, 4);
        var socket << udpBind("127.0.0.1", 0);
    `, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", FormatDiagnostics(diagnostics))
	}
	for _, dump := range []string{result.DumpHIR(), result.DumpMIR()} {
		for _, expected := range []string{"net_listener", "net_stream", "byte_array<u8>", "udp_socket"} {
			if !strings.Contains(dump, expected) {
				t.Fatalf("network dump missing %q:\n%s", expected, dump)
			}
		}
	}
}
