package types

import (
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZumbra0143DesktopMediaAndProcessBuiltinsAreTyped(t *testing.T) {
	p := parser.New(lexer.New(`
        var app << desktopApp({"backend": "headless"});
        var window << desktopWindow(app, {"title": "Media", "width": 2, "height": 2});
        var pixels << bytes(16);
        var presented << desktopWindowPresentRGBA(window, pixels, 2, 2);
        var key << desktopKeyDown(app, 29);
        var pad << desktopGamepadButton(app, 1, 0);
        var queued << desktopAudioQueue(app, bytes(4), 80, false);
        var queuedNow << desktopAudioQueued(app);
        var args << processArgs();
        var timestamp << unixTimeSeconds();
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{DesktopApp, DesktopWindow, ByteArray, Bool, Bool, Bool, Int, Int, Array, U64}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s, got %s", index, kind, got)
		}
	}
}
