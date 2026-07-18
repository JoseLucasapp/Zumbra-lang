package evaluator

import (
	"strings"
	"testing"

	"zumbra/object"
)

func TestFixedIntegerEvaluation(t *testing.T) {
	tests := []struct {
		input    string
		kind     object.FixedIntegerKind
		expected uint64
	}{
		{"255u8 + 1", object.FixedU8, 0},
		{"0u8 - 1", object.FixedU8, 255},
		{"bnot 0u8", object.FixedU8, 255},
		{"255u8 shl 1", object.FixedU8, 254},
		{"u16(0x8000)", object.FixedU16, 32768},
		{"wrapAdd(255u8, 1u8)", object.FixedU8, 0},
		{"satAdd(255u8, 1u8)", object.FixedU8, 255},
	}

	for _, test := range tests {
		result := testEval(test.input)
		value, ok := result.(*object.FixedInteger)
		if !ok {
			t.Fatalf("%s: expected fixed integer, got %T (%v)", test.input, result, result)
		}
		if value.Kind != test.kind || value.UnsignedValue() != test.expected {
			t.Fatalf("%s: expected %s raw=%d, got %s raw=%d", test.input, test.kind, test.expected, value.Kind, value.UnsignedValue())
		}
	}
}

func TestCheckedFixedIntegerOverflowReturnsError(t *testing.T) {
	result := testEval("checkedAdd(255u8, 1u8)")
	errorObject, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", result)
	}
	if !strings.Contains(errorObject.Message, "u8 overflow") {
		t.Fatalf("unexpected error: %s", errorObject.Message)
	}
}

func TestFixedIntegerTypeMismatchReturnsError(t *testing.T) {
	result := testEval("1u8 + 1u16")
	errorObject, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", result)
	}
	if !strings.Contains(errorObject.Message, "fixed integer types must match") {
		t.Fatalf("unexpected error: %s", errorObject.Message)
	}
}
