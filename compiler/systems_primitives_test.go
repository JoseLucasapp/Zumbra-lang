package compiler

import (
	"testing"

	"zumbra/code"
	"zumbra/object"
)

func TestCompileSystemIntegerLiterals(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "0xFF",
			expectedConstants: []interface{}{255},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "0b1010 + 0o10 + 1_000",
			expectedConstants: []interface{}{10, 8, 1000},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpAdd),
				code.Make(code.OpConstant, 2),
				code.Make(code.OpAdd),
				code.Make(code.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestCompileBitwiseExpressions(t *testing.T) {
	tests := []compilerTestCase{
		{
			input:             "1 band 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpBitAnd),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "1 bor 2 bxor 3",
			expectedConstants: []interface{}{1, 2, 3},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpConstant, 2),
				code.Make(code.OpBitXor),
				code.Make(code.OpBitOr),
				code.Make(code.OpPop),
			},
		},
		{
			input:             "bnot 1 shl 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpBitNot),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpShiftLeft),
				code.Make(code.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func TestCompileFloatLiteralWithSeparators(t *testing.T) {
	program := parse("10_000.25")
	compiled := New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}

	constants := compiled.Bytecode().Constants
	if len(constants) != 1 {
		t.Fatalf("expected one constant, got %d", len(constants))
	}
	value, ok := constants[0].(*object.Float)
	if !ok {
		t.Fatalf("expected float constant, got %T", constants[0])
	}
	if value.Value != 10000.25 {
		t.Fatalf("expected 10000.25, got %f", value.Value)
	}
}
