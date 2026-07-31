package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ14UITypesAndMethods(t *testing.T) {
	p := parser.New(lexer.New(`
        var app << desktopApp({"backend": "headless"});
        var window << app.window({"title": "UI", "width": 640, "height": 480});
        var state << uiState("ready");
        var label << uiText({"text": "Label"}, []);
        var root << uiColumn({}, [label]);
        var context << uiMount(app, window, root, {"theme": uiTheme("dark")});
        var snapshot << context.snapshot();
        var current << state.get();
        var found << context.find("missing");
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{DesktopApp, DesktopWindow, UIState, UINode, UINode, UIContext, Dict, Unknown, Unknown}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s got %s", index, kind, got)
		}
	}
}

func TestZ14UITypeErrors(t *testing.T) {
	p := parser.New(lexer.New(`
        uiMount("app", "window", "root", {});
        uiBind("node", "text", "state");
        uiCanvasCommand("canvas", "line", {});
        uiSetTheme("context", "theme");
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 {
		t.Fatalf("parser: %v", p.Errors())
	}
	if len(diagnostics) < 4 {
		t.Fatalf("expected type errors, got %v", diagnostics)
	}
}
