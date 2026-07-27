package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ12SQLiteBuiltinAndMethodTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var db << sqliteMemory();
        var result << db.exec("create table test(id integer)", []);
        var statement << db.prepare("insert into test(id) values (?)");
        var inserted << statement.exec([1]);
        var transaction << db.begin();
        var rows << transaction.query("select id from test", []);
        var active << transaction.active();
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{SQLiteDatabase, Dict, SQLiteStatement, Dict, SQLiteTransaction, Array, Bool}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s, got %s", index, kind, got)
		}
	}
}

func TestZ12SQLiteTypeErrorsAreRejected(t *testing.T) {
	p := parser.New(lexer.New(`
        var db << sqliteOpen(42);
        sqliteExec(db, "select ?", "not-an-array");
        sqliteCommit(db);
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 {
		t.Fatalf("parser diagnostics: %v", p.Errors())
	}
	if len(diagnostics) < 3 {
		t.Fatalf("expected SQLite type errors, got %v", diagnostics)
	}
}

func TestZ12SQLiteParameterArraysAllowMixedValuesContextually(t *testing.T) {
	p := parser.New(lexer.New(`
        var db << sqliteMemory();
        db.exec("create table users(name text, score integer)", []);
        db.exec("insert into users(name, score) values (?, ?)", ["Lucas", 42]);
        var statement << db.prepare("insert into users(name, score) values (?, ?)");
        statement.exec(["Zumbra", 100]);
        var transaction << db.begin();
        transaction.exec("insert into users(name, score) values (?, ?)", ["Temporary", 1]);
        transaction.rollback();
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("mixed SQL parameters should be accepted contextually: %v %v", p.Errors(), diagnostics)
	}
}
