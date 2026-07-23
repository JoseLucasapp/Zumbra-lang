package evaluator

import (
	"testing"
	"zumbra/object"
)

func TestEvaluatorSpawnAwaitAndChannel(t *testing.T) {
	result := testEval(`
        fct produce(messages) { send(messages, 7); closeChannel(messages); return; }
        var messages << channel(1);
        var producer << spawn produce(messages);
        var value << receive(messages);
        await producer;
        value;
    `)
	testIntegerObject(t, result, 7)
}

func TestEvaluatorAsyncFunctionReturnsTask(t *testing.T) {
	result := testEval(`
        var answer << async fct() { 42; };
        var task << answer();
        await task;
    `)
	testIntegerObject(t, result, 42)
}

func TestEvaluatorTaskTimeout(t *testing.T) {
	result := testEval(`
        fct slow() { sleepMs(20); 42; }
        var task << spawn slow();
        var timed << joinTimeout(task, 1);
        timed[1];
    `)
	value, ok := result.(*object.Boolean)
	if !ok || value.Value {
		t.Fatalf("expected timeout false, got %T %v", result, result)
	}
}
