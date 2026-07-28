package types

import (
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ13DesktopObjectsAndMethodsAreTyped(t *testing.T) {
	p := parser.New(lexer.New(`
        var app << desktopApp({"backend": "headless"});
        var window << app.window({"title": "Test", "width": 640, "height": 480});
        var tray << app.tray({"tooltip": "Test"});
        var process << desktopSpawn("/bin/sh", {"args": ["-c", "exit 0"]});
        var size << window.size();
        var paths << app.paths();
        var running << process.running();
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{DesktopApp, DesktopWindow, DesktopTray, DesktopProcess, Dict, Dict, Bool}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s, got %s", index, kind, got)
		}
	}
}

func TestZ13DesktopCallbacksUseContextualEventType(t *testing.T) {
	p := parser.New(lexer.New(`
        var app << desktopApp({"backend": "headless"});
        app.on("custom", fct(event) { event["type"]; });
        app.shortcut("ctrl+s", fct(event) { event["shortcut"]; });
        var tray << app.tray({"tooltip": "Test"});
        tray.add("quit", "Quit", fct(event) { event["id"]; });
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("desktop callbacks should infer dictionary events: %v %v", p.Errors(), diagnostics)
	}
}

func TestZ13DesktopTypeErrorsAreRejected(t *testing.T) {
	p := parser.New(lexer.New(`
        desktopApp("headless");
        desktopWindow("not-app", {});
        desktopSetClipboard(42, "text");
        desktopWindowSetSize("not-window", 800, 600);
        desktopSpawn(42, {});
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 {
		t.Fatalf("parser diagnostics: %v", p.Errors())
	}
	if len(diagnostics) < 5 {
		t.Fatalf("expected desktop type errors, got %v", diagnostics)
	}
}
