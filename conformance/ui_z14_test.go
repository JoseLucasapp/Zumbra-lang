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

func TestZ14UIHeadlessMatchesEvaluatorAndVM(t *testing.T) {
	source := `
        var app << desktopApp({"backend": "headless", "quitOnLastWindow": false});
        var window << app.window({"title": "UI", "width": 320, "height": 200});
        var state << uiState("ready");
        var label << uiText({"text": ""}, []);
        uiBind(label, "text", state);
        var clicked << false;
        var button << uiButton({"id": "run", "text": "Run", "onClick": fct(event) { event; clicked << true; uiStateSet(state, "clicked"); }}, []);
        var root << uiColumn({"padding": 8}, [button, label]);
        var context << uiMount(app, window, root, {"theme": uiTheme("light")});
        uiDispatch(context, {"type": "mouse_down", "x": 12, "y": 12});
        uiDispatch(context, {"type": "mouse_up", "x": 12, "y": 12});
        var snapshot << uiSnapshot(context);
        var result << toString(clicked) + ":" + uiGet(label, "text") + ":" + toString(snapshot["width"]);
        uiUnmount(context); app.close(); result;
    `
	result, diagnostics := pipeline.Build("z14-ui.zum", source, pipeline.Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline: %s", pipeline.FormatDiagnostics(diagnostics))
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
		t.Fatalf("evaluator=%s vm=%s", evaluated.Inspect(), fromVM.Inspect())
	}
	if evaluated.Inspect() != "true:clicked:320" {
		t.Fatalf("result=%q", evaluated.Inspect())
	}
}
