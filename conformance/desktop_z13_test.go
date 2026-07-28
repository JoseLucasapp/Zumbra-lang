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

func TestZ13DesktopHeadlessCallbacksMatchEvaluatorAndVM(t *testing.T) {
	source := `
        var app << desktopApp({"backend": "headless", "pollIntervalMs": 0});
        var window << app.window({"title": "Before", "width": 640, "height": 480});
        window.setTitle("After");
        window.setSize(960, 640);
        app.setClipboard("copied");
        var received << false;
        app.on("custom", fct(event) {
            received << event["message"] == "desktop-ok";
            app.quit();
        });
        app.emit({"type": "custom", "data": {"message": "desktop-ok"}});
        app.run();
        var size << window.size();
        var result << window.title() + ":" + toString(received) + ":" + app.clipboard() + ":" + toString(size["width"]);
        app.close();
        result;
    `
	result, diagnostics := pipeline.Build("z13-desktop.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %s", pipeline.FormatDiagnostics(diagnostics))
	}

	objectbuiltins.SetDesktopInvoker(func(handler object.Object, args ...object.Object) (object.Object, error) {
		return evaluator.InvokeFunction(handler, args)
	})
	evaluated := evaluator.EvalPipeline(result, object.NewEnvironment())

	compiled := compiler.New()
	if err := compiled.CompilePipeline(result); err != nil {
		t.Fatal(err)
	}
	globals := make([]object.Object, vm.GlobalSize)
	objectbuiltins.SetDesktopInvoker(func(handler object.Object, args ...object.Object) (object.Object, error) {
		return vm.InvokeFunction(handler, args, compiled.Bytecode().Constants, globals)
	})
	machine := vm.NewWithGlobalsStore(compiled.Bytecode(), globals)
	if err := machine.Run(); err != nil {
		t.Fatal(err)
	}
	fromVM := machine.LastPoppedStackElem()
	if evaluated.Type() != fromVM.Type() || evaluated.Inspect() != fromVM.Inspect() {
		t.Fatalf("evaluator=%s/%s vm=%s/%s", evaluated.Type(), evaluated.Inspect(), fromVM.Type(), fromVM.Inspect())
	}
	if evaluated.Inspect() != "After:true:copied:960" {
		t.Fatalf("unexpected result %q", evaluated.Inspect())
	}
}
