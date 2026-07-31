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

func TestSystemPrimitivesMatchEvaluatorAndVM(t *testing.T) {
	tests := []struct {
		source   string
		expected int64
	}{
		{"0xF0 band 0x0F", 0},
		{"0b1010 bor 0b0101", 15},
		{"0xFF bxor 0x0F", 240},
		{"1 shl 12", 4096},
		{"4096 shr 12", 1},
		{"bnot 0", -1},
		{"(0xABCD shr 8) band 0xFF", 171},
	}

	for _, test := range tests {
		evaluatorResult := runEvaluator(t, test.source)
		vmResult := runVM(t, test.source)

		if evaluatorResult != test.expected {
			t.Fatalf("evaluator %q: expected %d, got %d", test.source, test.expected, evaluatorResult)
		}
		if vmResult != test.expected {
			t.Fatalf("vm %q: expected %d, got %d", test.source, test.expected, vmResult)
		}
		if evaluatorResult != vmResult {
			t.Fatalf("runtime mismatch for %q: evaluator=%d vm=%d", test.source, evaluatorResult, vmResult)
		}
	}
}

func newEnvironment(t *testing.T) *object.Environment {
	t.Helper()
	return object.NewEnvironment()
}

func runEvaluator(t *testing.T, source string) int64 {
	t.Helper()
	lex := lexer.New(source)
	parsed := parser.New(lex)
	program := parsed.ParseProgram()
	if len(parsed.Errors()) != 0 {
		t.Fatalf("parser errors for %q: %v", source, parsed.Errors())
	}

	result := evaluator.Eval(program, newEnvironment(t))
	integer, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("evaluator %q returned %T", source, result)
	}
	return integer.Value
}

func runVM(t *testing.T, source string) int64 {
	t.Helper()
	lex := lexer.New(source)
	parsed := parser.New(lex)
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
		t.Fatalf("vm error for %q: %v", source, err)
	}

	result := machine.LastPoppedStackElem()
	integer, ok := result.(*object.Integer)
	if !ok {
		t.Fatalf("vm %q returned %T", source, result)
	}
	return integer.Value
}
