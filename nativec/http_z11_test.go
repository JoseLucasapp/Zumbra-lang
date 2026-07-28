package nativec_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"zumbra/nativec"
	"zumbra/pipeline"
)

func TestZ11HTTPAPIBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/http_api.zum")
	for _, expected := range []string{"200\nok\n0.6.0\nhello zumbra\n201\n42\nlocal\n", "true\nlocal-user\ntrue\n"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("native HTTP output missing %q:\n%s", expected, output)
		}
	}
	if !strings.Contains(output, "event: status") || !strings.Contains(output, "data: complete") {
		t.Fatalf("native SSE output is incomplete:\n%s", output)
	}
}

func TestZ11WebSocketBuildsAndRunsNatively(t *testing.T) {
	output := buildAndRunZ8(t, "code_examples/core/websocket.zum")
	if output != "text\necho:ping\nclose\ntrue\n" {
		t.Fatalf("unexpected native WebSocket output %q", output)
	}
}

func TestZ11NativeRuntimeIsConditionallyLinked(t *testing.T) {
	plain, diagnostics := pipeline.Build("plain.zum", `show(42);`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatal(pipeline.FormatDiagnostics(diagnostics))
	}
	plainSources, nativeDiagnostics := nativec.Generate(plain.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	if strings.Contains(string(plainSources.Runtime), "#define ZUMBRA_ENABLE_HTTP") {
		t.Fatal("sequential program unexpectedly enabled HTTP")
	}

	httpProgram, diagnostics := pipeline.Build("http.zum", `var app << httpApp();`, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatal(pipeline.FormatDiagnostics(diagnostics))
	}
	httpSources, nativeDiagnostics := nativec.Generate(httpProgram.MIR)
	if len(nativeDiagnostics) != 0 {
		t.Fatalf("native diagnostics: %#v", nativeDiagnostics)
	}
	runtimeSource := string(httpSources.Runtime)
	if !strings.Contains(runtimeSource, "#define ZUMBRA_ENABLE_HTTP 1") || !strings.Contains(runtimeSource, "#define ZUMBRA_ENABLE_NETWORK 1") {
		t.Fatal("HTTP program did not enable HTTP and network runtimes")
	}
}

func TestZ11HTTPSServerBuildsAndRunsNatively(t *testing.T) {
	certificate, err := filepath.Abs(filepath.Join("..", "code_examples", "native", "z10_test_cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := filepath.Abs(filepath.Join("..", "code_examples", "native", "z10_test_key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	source := fmt.Sprintf(`
        var app << httpApp();
        app.get("/", fct(request, response) {
            request;
            response;
            httpText(200, "secure");
        });
        var server << app.listenTls("127.0.0.1", 0, %q, %q);
        var reply << httpRequest("GET", "https://localhost:" + toString(server.port()) + "/", {}, "", 2000);
        show(httpStatus(reply));
        show(httpBody(reply));
        show(server.shutdown(2000));
    `, certificate, key)
	result := buildMIR(t, source)
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	directory := t.TempDir()
	output := filepath.Join(directory, "https-test")
	built, diagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release: true, Compiler: compiler, Output: output, BuildDir: filepath.Join(directory, "sources"),
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("native HTTPS build failed: err=%v diagnostics=%#v", err, diagnostics)
	}
	command := exec.Command(built.Output)
	command.Env = append(os.Environ(), "SSL_CERT_FILE="+certificate)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native HTTPS execution failed: %v\n%s", err, data)
	}
	if output := strings.ReplaceAll(string(data), "\r\n", "\n"); output != "200\nsecure\ntrue\n" {
		t.Fatalf("unexpected native HTTPS output %q", output)
	}
}

func TestZ11SecureWebSocketBuildsAndRunsNatively(t *testing.T) {
	certificate, err := filepath.Abs(filepath.Join("..", "code_examples", "native", "z10_test_cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	key, err := filepath.Abs(filepath.Join("..", "code_examples", "native", "z10_test_key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	source := fmt.Sprintf(`
        var app << httpApp();
        app.get("/socket", fct(request, response) {
            response;
            var socket << webSocketUpgrade(request);
            var message << webSocketRead(socket);
            webSocketWriteText(socket, "secure:" + message[1]);
            webSocketClose(socket, 1000, "complete");
            return;
        });
        var server << app.listenTls("127.0.0.1", 0, %q, %q);
        var socket << webSocketConnect("wss://localhost:" + toString(server.port()) + "/socket", {}, 2000);
        webSocketWriteText(socket, "ping");
        var response << webSocketRead(socket);
        show(response[1]);
        var closed << webSocketRead(socket);
        show(closed[0]);
        show(server.shutdown(2000));
    `, certificate, key)
	result := buildMIR(t, source)
	compiler, err := nativec.DetectCompiler("auto")
	if err != nil {
		t.Skip(err)
	}
	directory := t.TempDir()
	output := filepath.Join(directory, "wss-test")
	built, diagnostics, err := nativec.Build(result.MIR, nativec.BuildOptions{
		Release: true, Compiler: compiler, Output: output, BuildDir: filepath.Join(directory, "sources"),
	})
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("native WSS build failed: err=%v diagnostics=%#v", err, diagnostics)
	}
	command := exec.Command(built.Output)
	command.Env = append(os.Environ(), "SSL_CERT_FILE="+certificate)
	data, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native WSS execution failed: %v\n%s", err, data)
	}
	if output := strings.ReplaceAll(string(data), "\r\n", "\n"); output != "secure:ping\nclose\ntrue\n" {
		t.Fatalf("unexpected native WSS output %q", output)
	}
}
