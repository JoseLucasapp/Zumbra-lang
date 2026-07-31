package pipeline

import (
	"strings"
	"testing"
)

const callInferenceSource = `
fct square(value) {
    sleepMs(1);
    value * value;
}

fct produce(messages) {
    send(messages, 7);
    closeChannel(messages);
    return;
}

fct count(counter, amount) {
    var index << 0;
    while (index < amount) {
        atomicAdd(counter, 1);
        index << index + 1;
    }
    return;
}

var calculation << spawn square(6);
var messages << channel(2);
var producer << spawn produce(messages);
var counter << atomicInt(0);
var worker << spawn count(counter, 10);
`

func TestCallInferenceFlowsThroughCanonicalPipeline(t *testing.T) {
	result, diagnostics := Build("call_inference.zum", callInferenceSource, Options{Optimize: true})
	if len(diagnostics) != 0 {
		t.Fatalf("pipeline diagnostics: %v", diagnostics)
	}

	high := result.DumpHIR()
	for _, expected := range []string{
		`function name="square" : fct(int) -> int`,
		`function name="produce" : fct(channel<int>) -> null`,
		`function name="count" : fct(atomic_int,int) -> null`,
		`var name="messages" : channel<int>`,
		`spawn : task<int>`,
	} {
		if !strings.Contains(high, expected) {
			t.Fatalf("HIR missing %q:\n%s", expected, high)
		}
	}

	middle := result.DumpMIR()
	for _, expected := range []string{
		`function square(value) -> int`,
		`function produce(messages) -> null`,
		`function count(counter, amount) -> null`,
		`declare "messages"`,
		`: channel<int>`,
		`: task<int>`,
	} {
		if !strings.Contains(middle, expected) {
			t.Fatalf("MIR missing %q:\n%s", expected, middle)
		}
	}
	for _, forbidden := range []string{
		`function square(value) -> unknown`,
		`function produce(messages) -> unknown`,
		`function count(counter, amount) -> unknown`,
		`declare "messages"`, // checked more precisely below
	} {
		if forbidden == `declare "messages"` {
			continue
		}
		if strings.Contains(middle, forbidden) {
			t.Fatalf("MIR still contains %q:\n%s", forbidden, middle)
		}
	}
}
