package conformance

import (
	"testing"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestZ12SQLiteMatchesEvaluatorAndVM(t *testing.T) {
	source := `
        var db << sqliteMemory();
        db.exec("create table values_table(value integer)", []);
        db.exec("insert into values_table(value) values (?)", [42]);
        var rows << db.query("select value from values_table", []);
        var result << rows[0]["value"];
        db.close();
        result;
    `
	result, diagnostics := pipeline.Build("sqlite-conformance.zum", source, pipeline.Options{Optimize: true})
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
	if evaluated.Inspect() != "42" {
		t.Fatalf("unexpected result %s", evaluated.Inspect())
	}
}
