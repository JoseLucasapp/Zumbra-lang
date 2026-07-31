package conformance

import (
	"testing"

	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/object"
	"zumbra/pipeline"
	"zumbra/vm"
)

func requireZ17EvaluatorVMMatch(t *testing.T, name, source, expected string) {
	t.Helper()
	result, diagnostics := pipeline.Build(name+".zum", source, pipeline.Options{Optimize: true})
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
	if evaluated.Inspect() != expected {
		t.Fatalf("expected %q, got %q", expected, evaluated.Inspect())
	}
}

func TestZ17ExplicitMemoryMatchesEvaluatorAndVM(t *testing.T) {
	requireZ17EvaluatorVMMatch(t, "z17-memory", `
        memoryResetStats();
        var pointer << alloc("i32", 4);
        pointer[0] << 10i32;
        pointer[1] << 20i32;
        var borrowed << borrowPointer(pointer);
        var result << borrowed[1];
        releaseBorrow(borrowed);
        pointer << realloc(pointer, 8);
        pointer[7] << 80i32;
        result << result + pointer[7];
        free(pointer);
        result + memoryStats()["active_blocks"];
    `, "100")
}

func TestZ17ArenaAndAtomicsMatchEvaluatorAndVM(t *testing.T) {
	requireZ17EvaluatorVMMatch(t, "z17-arena-atomics", `
        var arenaValue << arenaCreate();
        var temporary << arenaAlloc(arenaValue, "u16", 2);
        temporary[0] << 7u16;
        var atomicValue << alloc("u64", 1);
        atomicPointerStore(atomicValue, 4u64);
        var sum << toInt(temporary[0]) + toInt(atomicPointerAdd(atomicValue, 3u64));
        free(atomicValue);
        arenaFree(arenaValue);
        sum;
    `, "14")
}

func TestZ17UnsafeRawPointerMatchesEvaluatorAndVM(t *testing.T) {
	requireZ17EvaluatorVMMatch(t, "z17-raw", `
        var owned << alloc("u8", 1);
        owned[0] << 55u8;
        var raw << nullPointer("u8");
        unsafe {
            raw << pointerFromAddress("u8", pointerAddress(owned), 1, true);
            raw[0] << 56u8;
        }
        var result << toInt(owned[0]);
        free(owned);
        result;
    `, "56")
}
