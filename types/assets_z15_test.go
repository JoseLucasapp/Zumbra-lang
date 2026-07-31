package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ15AssetBuiltinTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var exists << assetExists("assets/a.txt");
        var text << assetText("assets/a.txt");
        var raw << assetBytes("assets/a.bin");
        var names << assetList();
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{Bool, String, ByteArray, Array}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s got %s", index, kind, got)
		}
	}
}
