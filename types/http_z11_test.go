package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ11HTTPBuiltinAndMethodTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var app << httpApp();
        app.get("/health", fct(request, response) { httpJson(200, {"status": "ok"}); });
        var server << app.listen("127.0.0.1", 0);
        var response << httpRequest("GET", "http://127.0.0.1", {}, "", 1000);
        var parsed << jsonParse("42");
        var token << jwtSignHS256({"sub":"local"}, "secret", 60);
        var socket << webSocketConnect("ws://127.0.0.1:1/socket", {}, 1);
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	checks := map[int]Kind{0: HttpApp, 2: HttpServer, 3: HttpClientResponse, 4: Unknown, 5: String, 6: WebSocket}
	for index, kind := range checks {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s, got %s", index, kind, got)
		}
	}
}

func TestZ11HTTPTypeErrorsAreRejected(t *testing.T) {
	p := parser.New(lexer.New(`
        var app << httpApp();
        app.get(42, "not-a-handler");
        httpRequest("GET", 10, {}, "", "bad");
        webSocketWriteText(app, 42);
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 {
		t.Fatalf("parser diagnostics: %v", p.Errors())
	}
	if len(diagnostics) < 3 {
		t.Fatalf("expected HTTP type errors, got %v", diagnostics)
	}
}

func TestZ11HeterogeneousJSONDictWidensValueType(t *testing.T) {
	p := parser.New(lexer.New(`var payload << {"name":"zumbra", "value":42, "active":true};`))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	statement := program.Statements[0].(*ast.VarStatement)
	got := analysis.TypeOf(statement.Value)
	if got.Kind != Dict || got.Value == nil || got.Value.Kind != Unknown {
		t.Fatalf("expected dict<string, unknown>, got %s", got)
	}
}
