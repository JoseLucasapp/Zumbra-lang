package conformance

import (
	"testing"

	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestCallInferencePreservesEvaluatorAndVMBehavior(t *testing.T) {
	result, diagnostics := pipeline.Build("call-inference.zum", `
        var identity << fct(value) { value; };
        var messages << channel(1);
        fct publish(target, value) {
            send(target, value);
            return;
        }
        var producer << spawn publish(messages, identity(42));
        var answer << receive(messages);
        await producer;
        answer;
    `, pipeline.Options{Optimize: true})
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
