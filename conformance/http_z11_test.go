package conformance

import (
	"testing"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestZ11JSONAndJWTMatchEvaluatorAndVM(t *testing.T) {
	result, diagnostics := pipeline.Build("http-conformance.zum", `
        var encoded << jsonStringify({"name":"zumbra", "value":42});
        var decoded << jsonParse(encoded);
        var token << jwtSignHS256({"sub":"local"}, "secret", 60);
        var verified << jwtVerifyHS256(token, "secret");
        decoded["name"] + ":" + verified[1]["sub"];
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
	if evaluated.Inspect() != "zumbra:local" {
		t.Fatalf("unexpected result %s", evaluated.Inspect())
	}
}
