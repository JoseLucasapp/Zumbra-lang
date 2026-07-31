package vm

import "testing"

func TestVMSpawnAwaitChannelAndAtomic(t *testing.T) {
	tests := []vmTestCase{
		{input: `fct answer(){42;} var task << spawn answer(); await task;`, expected: 42},
		{input: `var answer << async fct(){21*2;}; await answer();`, expected: 42},
		{input: `fct produce(ch){send(ch,7);closeChannel(ch);return;} var ch<<channel(1);var task<<spawn produce(ch);var value<<receive(ch);await task;value;`, expected: 7},
		{input: `var value<<atomicInt(10);atomicAdd(value,5);atomicLoad(value);`, expected: 15},
	}
	runVmTests(t, tests)
}

func TestVMAsyncMethodReturnsTask(t *testing.T) {
	tests := []vmTestCase{{
		input:    `struct Worker { value: int; async fct calculate(amount) { self.value + amount; } } var worker << Worker(20); await worker.calculate(22);`,
		expected: 42,
	}}
	runVmTests(t, tests)
}
