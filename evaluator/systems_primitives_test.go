package evaluator

import (
	"strings"
	"testing"

	"zumbra/object"
)

func TestSystemIntegerLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0xFF", 255},
		{"0b1010", 10},
		{"0o755", 493},
		{"1_000_000", 1_000_000},
	}

	for _, test := range tests {
		testIntegerObject(t, testEval(test.input), test.expected)
	}
}

func TestBitwiseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"0b1100 band 0b1010", 0b1000},
		{"0b1100 bor 0b0011", 0b1111},
		{"0b1100 bxor 0b1010", 0b0110},
		{"1 shl 8", 256},
		{"256 shr 8", 1},
		{"bnot 0", -1},
		{"(0xF0 band 0x0F) bor 3", 3},
	}

	for _, test := range tests {
		testIntegerObject(t, testEval(test.input), test.expected)
	}
}

func TestBitwiseRejectsNonInteger(t *testing.T) {
	evaluated := testEval(`1 band 1.5`)
	errorObject, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", evaluated)
	}
	if !strings.Contains(errorObject.Message, "unknown operator") {
		t.Fatalf("unexpected error: %s", errorObject.Message)
	}
}

func TestShiftCountRange(t *testing.T) {
	evaluated := testEval("1 shl 64")
	errorObject, ok := evaluated.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", evaluated)
	}
	if !strings.Contains(errorObject.Message, "shift count must be between 0 and 63") {
		t.Fatalf("unexpected error: %s", errorObject.Message)
	}
}

func TestFloatLiteralWithSeparators(t *testing.T) {
	evaluated := testEval("10_000.25")
	value, ok := evaluated.(*object.Float)
	if !ok {
		t.Fatalf("expected float, got %T", evaluated)
	}
	if value.Value != 10000.25 {
		t.Fatalf("expected 10000.25, got %f", value.Value)
	}
}

func TestBangOperatorUsesFixedComparisonBooleanValue(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"!(0u8 band 0x02u8 != 0u8)", true},
		{"!(0x02u8 band 0x02u8 != 0u8)", false},
		{"!!(0u8 band 0x02u8 != 0u8)", false},
		{"(0u8 band 0x02u8 != 0u8) == false", true},
	}

	for _, test := range tests {
		evaluated := testEval(test.input)
		boolean, ok := evaluated.(*object.Boolean)
		if !ok {
			t.Fatalf("%s: expected boolean, got %T", test.input, evaluated)
		}
		if boolean.Value != test.expected {
			t.Fatalf("%s: expected %t, got %t", test.input, test.expected, boolean.Value)
		}
	}
}

func TestBangOperatorUsesBooleanValueStoredInStruct(t *testing.T) {
	evaluated := testEval(`
		struct Header { battery: bool; }
		var flags << 0u8;
		var parsed << Header((flags band 0x02u8) != 0u8);
		!parsed.battery;
	`)
	boolean, ok := evaluated.(*object.Boolean)
	if !ok {
		t.Fatalf("expected boolean, got %T", evaluated)
	}
	if !boolean.Value {
		t.Fatal("expected negated false struct field to be true")
	}
}
