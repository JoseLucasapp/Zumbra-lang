package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestZ8ModulesPreserveEvaluatorAndVMBehavior(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "math.zum"), []byte(`pub fct add(left, right) { left + right; }`), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "app.zum")
	if err := os.WriteFile(entry, []byte(`import "math.zum" as math; math.add(40, 2);`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, diagnostics := pipeline.BuildFile(entry, pipeline.Options{Optimize: true})
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
		t.Fatalf("expected 42, got %s", evaluated.Inspect())
	}
}
