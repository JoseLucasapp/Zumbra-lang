package types

import (
	"testing"

	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestZ12CompleteBuiltinAndObjectTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        var db << sqliteMemory();
        var rows << db.stream("select 1", {});
        var config << configFrom({"port": "8080"});
        var port << config.int("port", 0);
        var registry << metrics();
        var span << traceStart("request", {});
        var store << sessionSQLite(":memory:");
        var limiter << rateLimiter(2, 1000);
        var encoded << binaryEncode({"ok": true});
        var decoded << binaryDecode(encoded);
        var postgres << postgresOpen("postgres://localhost/test", {"maxOpen": 4});
        var statement << postgres.prepare("select $1");
        var redis << redisOpen("127.0.0.1", 6379, "", 0, 4);
    `))
	program := p.ParseProgram()
	analysis, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), diagnostics)
	}
	expected := []Kind{
		SQLiteDatabase, SQLRows, Config, Int, MetricsRegistry, TraceSpan,
		SessionStore, RateLimiter, ByteArray, Unknown, PostgresDatabase,
		PostgresStatement, RedisClient,
	}
	for index, kind := range expected {
		statement := program.Statements[index].(*ast.VarStatement)
		if got := analysis.TypeOf(statement.Value); got.Kind != kind {
			t.Fatalf("statement %d expected %s, got %s", index, kind, got)
		}
	}
}

func TestZ12NamedSQLParametersAreContextualOnly(t *testing.T) {
	valid := parser.New(lexer.New(`
        var db << sqliteMemory();
        db.exec("create table users(name text, score integer)", {});
        db.exec("insert into users(name, score) values (:name, :score)", {"name": "Lucas", "score": 42});
        var tx << db.begin();
        tx.exec("update users set score = :score where name = :name", {"score": 43, "name": "Lucas"});
        tx.commit();
    `))
	program := valid.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(valid.Errors()) != 0 || len(diagnostics) != 0 {
		t.Fatalf("named SQL parameters should be valid: %v %v", valid.Errors(), diagnostics)
	}

	invalid := parser.New(lexer.New(`
        var ordinary << ["Lucas", 42, true];
    `))
	program = invalid.ParseProgram()
	_, diagnostics = AnalyzeWithInfo(program)
	if len(invalid.Errors()) != 0 {
		t.Fatalf("parser diagnostics: %v", invalid.Errors())
	}
	if len(diagnostics) == 0 {
		t.Fatal("mixed ordinary array was accepted outside SQL context")
	}
}

func TestZ12ServiceTypeErrorsAreRejected(t *testing.T) {
	p := parser.New(lexer.New(`
        configLoad(42);
        rateLimiter("two", 1000);
        sessionRedis("not-client", "prefix");
        postgresOpen(42, {});
        redisOpen("localhost", "6379", "", 0, 4);
        binaryDecode("not-bytes");
    `))
	program := p.ParseProgram()
	_, diagnostics := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 {
		t.Fatalf("parser diagnostics: %v", p.Errors())
	}
	if len(diagnostics) < 6 {
		t.Fatalf("expected service type errors, got %v", diagnostics)
	}
}
