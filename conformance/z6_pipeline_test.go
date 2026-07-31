package conformance

import (
	"testing"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestZ6PipelinePreservesEvaluatorAndVMBehavior(t *testing.T) {
	source := `
struct Counter { value: int; fct add(amount) { self.value << self.value + amount; } }
var counter << Counter(10);
counter.add(4);
var folded << 2 + 3 * 4;
counter.value + folded;
`
	result, diagnostics := pipeline.Build("test.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %v", diagnostics)
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
	if evaluated.Inspect() != "28" {
		t.Fatalf("expected 28, got %s", evaluated.Inspect())
	}
}
