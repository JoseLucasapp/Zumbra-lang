package conformance

import (
	"testing"

	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func requireZ12EvaluatorVMMatch(t *testing.T, name, source, expected string) {
	t.Helper()
	result, diagnostics := pipeline.Build(name+".zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}
	evaluated := evaluator.EvalPipeline(result, object.NewEnvironment())
	compiled := compiler.New()
	if err := compiled.CompilePipeline(result); err != nil {
		t.Fatal(err)
	}
	machine := vm.New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}
	fromVM := machine.LastPoppedStackElem()
	if evaluated.Type() != fromVM.Type() || evaluated.Inspect() != fromVM.Inspect() {
		t.Fatalf("evaluator=%s/%s vm=%s/%s", evaluated.Type(), evaluated.Inspect(), fromVM.Type(), fromVM.Inspect())
	}
	if evaluated.Inspect() != expected {
		t.Fatalf("expected %q, got %q", expected, evaluated.Inspect())
	}
}

func TestZ12AdvancedSQLiteMatchesEvaluatorAndVM(t *testing.T) {
	requireZ12EvaluatorVMMatch(t, "z12-sqlite-advanced", `
        var db << sqliteMemory();
        sqliteMigrate(db, [
            {"version": 1, "name": "users", "sql": "create table users(id integer primary key, name text, score integer)"}
        ]);
        sqliteExec(db, "insert into users(name, score) values (:name, :score)", {"name": "Lucas", "score": 42});
        var tx << sqliteBegin(db);
        sqliteSavepoint(tx, "before_bonus");
        sqliteTransactionExec(tx, "update users set score = score + 100 where name = :name", {"name": "Lucas"});
        sqliteRollbackTo(tx, "before_bonus");
        sqliteRelease(tx, "before_bonus");
        sqliteCommit(tx);
        var row << sqliteQueryOne(db, "select score from users where name = :name", {"name": "Lucas"});
        var result << row["score"] + sqliteSchemaVersion(db);
        sqliteClose(db);
        result;
    `, "43")
}

func TestZ12ConfigurationMetricsTracingAndRateLimitMatchEvaluatorAndVM(t *testing.T) {
	requireZ12EvaluatorVMMatch(t, "z12-services", `
        var config << configFrom({"port": "8080", "debug": "true", "secret": "token"});
        configSecret(config, "secret");
        var registry << metrics();
        metricsCounter(registry, "requests", 2, {"route": "/"});
        var span << traceStart("request", {"route": "/"});
        traceEvent(span, "handled", {"status": 200});
        var finished << traceFinish(span, "ok");
        var limiter << rateLimiter(1, 1000);
        var first << rateAllow(limiter, "client");
        var second << rateAllow(limiter, "client");
        configInt(config, "port", 0) + metricsSnapshot(registry)["counters"]["requests{route=/}"] + (first["remaining"] - second["remaining"]);
    `, "8082")
}

func TestZ12BinarySerializationMatchesEvaluatorAndVM(t *testing.T) {
	requireZ12EvaluatorVMMatch(t, "z12-serialization", `
        var raw << bytes(2);
        raw[0] << 0xABu8;
        raw[1] << 0xCDu8;
        var encoded << binaryEncode({"score": 42, "raw": raw});
        var decoded << binaryDecode(encoded);
        decoded["score"] + toInt(readU16BE(decoded["raw"], 0));
    `, "44023")
}
