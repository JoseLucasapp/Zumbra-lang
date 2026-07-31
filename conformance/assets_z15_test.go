package conformance

import (
	"testing"
	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	objectbuiltins "zumbra/object/builtins"
	"zumbra/pipeline"
	"zumbra/vm"
)

func TestZ15EmbeddedAssetsMatchEvaluatorAndVM(t *testing.T) {
	defer objectbuiltins.ResetEmbeddedAssets()
	if err := objectbuiltins.ConfigureEmbeddedAssets([]objectbuiltins.EmbeddedAsset{{Name: "assets/hello.txt", Data: []byte("olá")}}); err != nil {
		t.Fatal(err)
	}
	source := `assetText("assets/hello.txt") + ":" + toString(assetExists("assets/hello.txt")) + ":" + toString(sizeOf(assetList()));`
	result, diagnostics := pipeline.Build("z15-assets.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
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
		t.Fatalf("evaluator=%s vm=%s", evaluated.Inspect(), fromVM.Inspect())
	}
	if evaluated.Inspect() != "olá:true:1" {
		t.Fatalf("result=%q", evaluated.Inspect())
	}
}
