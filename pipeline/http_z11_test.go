package pipeline

import (
	"strings"
	"testing"
)

func TestZ11HTTPTypesFlowThroughHIRAndMIR(t *testing.T) {
	result, diagnostics := Build("http-types.zum", `
        var app << httpApp();
        app.get("/health", fct(request, response) {
            request;
            response;
            httpJson(200, {"status":"ok"});
        });
        var server << app.listen("127.0.0.1", 0);
        var clientResponse << httpRequest("GET", "http://127.0.0.1", {}, "", 1000);
        httpShutdown(server, 1000);
    `, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", FormatDiagnostics(diagnostics))
	}
	for _, dump := range []string{result.DumpHIR(), result.DumpMIR()} {
		for _, expected := range []string{"http_app", "http_server", "http_client_response", "http_response"} {
			if !strings.Contains(dump, expected) {
				t.Fatalf("HTTP dump missing %q:\n%s", expected, dump)
			}
		}
	}
}

func TestZ11WebSocketTypeFlowsThroughPipeline(t *testing.T) {
	result, diagnostics := Build("websocket-types.zum", `
        var socket << webSocketConnect("ws://127.0.0.1:1/socket", {}, 1);
        var frame << webSocketReadTimeout(socket, 1);
    `, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", FormatDiagnostics(diagnostics))
	}
	if !strings.Contains(result.DumpHIR(), "web_socket") || !strings.Contains(result.DumpMIR(), "web_socket") {
		t.Fatalf("WebSocket type missing:\nHIR:\n%s\nMIR:\n%s", result.DumpHIR(), result.DumpMIR())
	}
}
