package conformance

import (
	"testing"

	"zumbra/compiler"
	"zumbra/evaluator"
	"zumbra/lexer"
	"zumbra/object"
	"zumbra/parser"
	"zumbra/vm"
)

func TestFixedIntegersMatchEvaluatorAndVM(t *testing.T) {
	tests := []string{
		"255u8 + 1",
		"0u16 - 1",
		"bnot 0u8",
		"i8(-128) shr 1",
		"wrapMul(200u8, 2u8)",
		"satAdd(255u8, 1u8)",
		"u32(0xFFFF) bor 0x10000",
	}

	for _, source := range tests {
		evaluatorValue := runFixedEvaluator(t, source)
		vmValue := runFixedVM(t, source)
		if evaluatorValue.Kind != vmValue.Kind || evaluatorValue.UnsignedValue() != vmValue.UnsignedValue() {
			t.Fatalf(
				"runtime mismatch for %q: evaluator=%s/%d VM=%s/%d",
				source,
				evaluatorValue.Kind,
				evaluatorValue.UnsignedValue(),
				vmValue.Kind,
				vmValue.UnsignedValue(),
			)
		}
	}
}

func runFixedEvaluator(t *testing.T, source string) *object.FixedInteger {
	t.Helper()
	parsed := parser.New(lexer.New(source))
	program := parsed.ParseProgram()
	if len(parsed.Errors()) != 0 {
		t.Fatalf("parser errors for %q: %v", source, parsed.Errors())
	}
	result := evaluator.Eval(program, object.NewEnvironment())
	value, ok := result.(*object.FixedInteger)
	if !ok {
		t.Fatalf("evaluator %q returned %T (%v)", source, result, result)
	}
	return value
}

func runFixedVM(t *testing.T, source string) *object.FixedInteger {
	t.Helper()
	parsed := parser.New(lexer.New(source))
	program := parsed.ParseProgram()
	if len(parsed.Errors()) != 0 {
		t.Fatalf("parser errors for %q: %v", source, parsed.Errors())
	}
	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error for %q: %v", source, err)
	}
	machine := vm.New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("VM error for %q: %v", source, err)
	}
	result := machine.LastPoppedStackElem()
	value, ok := result.(*object.FixedInteger)
	if !ok {
		t.Fatalf("VM %q returned %T (%v)", source, result, result)
	}
	return value
}
