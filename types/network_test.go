package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestNetworkBuiltinTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var listener << tcpListen("127.0.0.1", 0);
        var connection << tcpConnectTimeout("127.0.0.1", listenerPort(listener), 1000);
        var packet << streamRead(connection, 64);
        var udp << udpBind("127.0.0.1", 0);
        var addresses << dnsLookup("localhost");
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{NetListener, NetStream, ByteArray, UDPSocket, Array}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s, got %s", index, kind, got)
		}
	}
}

func TestNetworkTypeErrorsAreRejected(t *testing.T) {
	p := parser.New(lexer.New(`
        var listener << tcpListen(10, "bad");
        streamWriteAll(listener, 42);
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 {
		t.Fatalf("parser diagnostics: %v", p.Errors())
	}
	if len(diagnostics) < 2 {
		t.Fatalf("expected network type errors, got %v", diagnostics)
	}
}
