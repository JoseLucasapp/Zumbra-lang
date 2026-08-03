package vm

import (
	"strings"
	"testing"

	"zumbra/ast"
	"zumbra/compiler"
	"zumbra/object"
)

func TestSystemIntegerLiteralsOnVM(t *testing.T) {
	tests := []vmTestCase{
		{"0xFF", 255},
		{"0b1010", 10},
		{"0o755", 493},
		{"1_000_000", 1_000_000},
	}
	runVmTests(t, tests)
}

func TestBitwiseExpressionsOnVM(t *testing.T) {
	tests := []vmTestCase{
		{"0b1100 band 0b1010", 8},
		{"0b1100 bor 0b0011", 15},
		{"0b1100 bxor 0b1010", 6},
		{"1 shl 8", 256},
		{"256 shr 8", 1},
		{"bnot 0", -1},
	}
	runVmTests(t, tests)
}

func TestVMRejectsInvalidShiftCount(t *testing.T) {
	program := parse("1 shl 64")
	compiler := newCompilerForSystemPrimitiveTest(t, program)
	machine := New(compiler.Bytecode())
	if err := machine.Run(); err == nil || !strings.Contains(err.Error(), "shift count must be between 0 and 63") {
		t.Fatalf("expected shift count error, got %v", err)
	}
}

func newCompilerForSystemPrimitiveTest(t *testing.T, program *ast.Program) *compiler.Compiler {
	t.Helper()
	compiled := compiler.New()
	if err := compiled.Compile(program); err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	return compiled
}

func TestFloatLiteralWithSeparatorsOnVM(t *testing.T) {
	program := parse("10_000.25")
	compiled := newCompilerForSystemPrimitiveTest(t, program)
	machine := New(compiled.Bytecode())
	if err := machine.Run(); err != nil {
		t.Fatalf("vm error: %s", err)
	}

	value, ok := machine.LastPoppedStackElem().(*object.Float)
	if !ok {
		t.Fatalf("expected float, got %T", machine.LastPoppedStackElem())
	}
	if value.Value != 10000.25 {
		t.Fatalf("expected 10000.25, got %f", value.Value)
	}
}

func TestBangOperatorUsesFixedComparisonBooleanValueOnVM(t *testing.T) {
	tests := []vmTestCase{
		{"!(0u8 band 0x02u8 != 0u8)", true},
		{"!(0x02u8 band 0x02u8 != 0u8)", false},
		{"!!(0u8 band 0x02u8 != 0u8)", false},
		{"(0u8 band 0x02u8 != 0u8) == false", true},
		{`
			struct Header { battery: bool; }
			var flags << 0u8;
			var parsed << Header((flags band 0x02u8) != 0u8);
			!parsed.battery;
		`, true},
	}
	runVmTests(t, tests)
}
