package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestAnalyzeWithInfoRecordsExpressionTypes(t *testing.T) {
	p := parser.New(lexer.New(`var value << 1u8 + 2u8;`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	analysis, errs := AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type errors: %v", errs)
	}
	stmt := program.Statements[0].(*ast.VarStatement)
	if got := analysis.TypeOf(stmt.Value).String(); got != "u8" {
		t.Fatalf("expected u8, got %s", got)
	}
	if got, ok := analysis.Global("value"); !ok || got.String() != "u8" {
		t.Fatalf("expected global value:u8, got %v %v", got, ok)
	}
}

func TestAnalyzeWithInfoRecordsMethodAndMatchTypes(t *testing.T) {
	p := parser.New(lexer.New(`
struct Box { value: int; fct get() { self.value; } }
enum Mode { A; B; }
var mode << Mode.A;
var label << match(mode) { case Mode.A { "a"; } else { "b"; } };
`))
	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	analysis, errs := AnalyzeWithInfo(program)
	if len(errs) != 0 {
		t.Fatalf("type errors: %v", errs)
	}
	structStmt := program.Statements[0].(*ast.StructStatement)
	if got := analysis.TypeOf(structStmt.Methods[0].Function).String(); got != "fct() -> int" {
		t.Fatalf("unexpected method type %s", got)
	}
	label := program.Statements[3].(*ast.VarStatement)
	if got := analysis.TypeOf(label.Value).String(); got != "string" {
		t.Fatalf("unexpected match type %s", got)
	}
}
