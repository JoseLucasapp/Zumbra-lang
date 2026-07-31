package conformance

import (
	"testing"

	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestTCPNetworkingMatchesEvaluatorAndVM(t *testing.T) {
	result, diagnostics := pipeline.Build("network-conformance.zum", `
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
        sizeOf(response);
    `, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}

	evaluated := evaluator.EvalPipeline(result, object.NewEnvironment())
	compiled := compiler.New()
	if err := compiled.CompilePipeline(result); err != nil {
		t.Fatal(err)
	}
	machine := vm.New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}
	fromVM := machine.LastPoppedStackElem()
	if evaluated.Type() != fromVM.Type() || evaluated.Inspect() != fromVM.Inspect() {
		t.Fatalf("evaluator=%s/%s vm=%s/%s", evaluated.Type(), evaluated.Inspect(), fromVM.Type(), fromVM.Inspect())
	}
	if evaluated.Inspect() != "4" {
		t.Fatalf("expected 4, got %s", evaluated.Inspect())
	}
}
