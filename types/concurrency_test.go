package types

import (
	"testing"
	"zumbra/ast"
	"zumbra/lexer"
	"zumbra/parser"
)

func TestSpawnAndAwaitTypes(t *testing.T) {
	p := parser.New(lexer.New(`
        fct answer() { 42; }
        var task << spawn answer();
        var result << await task;
    `))
	program := p.ParseProgram()
	analysis, errs := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), errs)
	}
	taskStatement := program.Statements[1].(*ast.VarStatement)
	resultStatement := program.Statements[2].(*ast.VarStatement)
	taskType := analysis.TypeOf(taskStatement.Value)
	if taskType.Kind != Task || taskType.Elem == nil || taskType.Elem.Kind != Int {
		t.Fatalf("unexpected task type: %s", taskType)
	}
	if result := analysis.TypeOf(resultStatement.Value); result.Kind != Int {
		t.Fatalf("unexpected await type: %s", result)
	}
}

func TestChannelElementTypeIsInferredAndEnforced(t *testing.T) {
	p := parser.New(lexer.New(`
        var messages << channel(2);
        send(messages, "ready");
        var value << receive(messages);
    `))
	program := p.ParseProgram()
	analysis, errs := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), errs)
	}
	valueStatement := program.Statements[2].(*ast.VarStatement)
	if got := analysis.TypeOf(valueStatement.Value); got.Kind != String {
		t.Fatalf("expected receive to infer string, got %s", got)
	}

	invalid := parser.New(lexer.New(`
        var messages << channel(2);
        send(messages, "ready");
        send(messages, 42);
    `))
	invalidProgram := invalid.ParseProgram()
	_, invalidErrors := AnalyzeWithInfo(invalidProgram)
	if len(invalid.Errors()) != 0 {
		t.Fatalf("parser errors: %v", invalid.Errors())
	}
	if len(invalidErrors) == 0 {
		t.Fatal("expected mismatched channel element type to be rejected")
	}
}

func TestAsyncFunctionCallReturnsTaskOfBodyResult(t *testing.T) {
	p := parser.New(lexer.New(`
        var answer << async fct() { 42; };
        var task << answer();
        var result << await task;
    `))
	program := p.ParseProgram()
	analysis, errs := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), errs)
	}
	functionStatement := program.Statements[0].(*ast.VarStatement)
	taskStatement := program.Statements[1].(*ast.VarStatement)
	resultStatement := program.Statements[2].(*ast.VarStatement)
	functionType := analysis.TypeOf(functionStatement.Value)
	if functionType.Kind != Func || !functionType.Async || functionType.Return == nil || functionType.Return.Kind != Int {
		t.Fatalf("unexpected async function type: %s", functionType)
	}
	taskType := analysis.TypeOf(taskStatement.Value)
	if taskType.Kind != Task || taskType.Elem == nil || taskType.Elem.Kind != Int {
		t.Fatalf("unexpected async call type: %s", taskType)
	}
	if resultType := analysis.TypeOf(resultStatement.Value); resultType.Kind != Int {
		t.Fatalf("unexpected awaited type: %s", resultType)
	}
}

func TestSpawnAsyncFunctionDoesNotCreateNestedTask(t *testing.T) {
	p := parser.New(lexer.New(`
        var answer << async fct() { 42; };
        var task << spawn answer();
    `))
	program := p.ParseProgram()
	analysis, errs := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), errs)
	}
	taskStatement := program.Statements[1].(*ast.VarStatement)
	taskType := analysis.TypeOf(taskStatement.Value)
	if taskType.Kind != Task || taskType.Elem == nil || taskType.Elem.Kind != Int {
		t.Fatalf("spawn async must return task<int>, got %s", taskType)
	}
}

func TestAsyncMethodCallReturnsTask(t *testing.T) {
	p := parser.New(lexer.New(`
        struct Worker {
            value: int;
            async fct calculate() { self.value; }
        }
        var worker << Worker(42);
        var task << worker.calculate();
        var result << await task;
    `))
	program := p.ParseProgram()
	analysis, errs := AnalyzeWithInfo(program)
	if len(p.Errors()) != 0 || len(errs) != 0 {
		t.Fatalf("diagnostics: %v %v", p.Errors(), errs)
	}
	taskStatement := program.Statements[2].(*ast.VarStatement)
	resultStatement := program.Statements[3].(*ast.VarStatement)
	if got := analysis.TypeOf(taskStatement.Value); got.Kind != Task || got.Elem == nil || got.Elem.Kind != Int {
		t.Fatalf("unexpected async method task type: %s", got)
	}
	if got := analysis.TypeOf(resultStatement.Value); got.Kind != Int {
		t.Fatalf("unexpected async method result type: %s", got)
	}
}
