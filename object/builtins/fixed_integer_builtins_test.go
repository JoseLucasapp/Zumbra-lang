package builtins

import (
	"strings"
	"testing"

	"zumbra/numeric"
	"zumbra/object"
)

func TestFixedIntegerConversionBuiltin(t *testing.T) {
	builtin := FixedIntegerConversionBuiltin(object.FixedU8)
	result := builtin.Fn(&object.Integer{Value: 255})
	value, ok := result.(*object.FixedInteger)
	if !ok {
		t.Fatalf("expected fixed integer, got %T", result)
	}
	if value.Kind != object.FixedU8 || value.UnsignedValue() != 255 {
		t.Fatalf("unexpected value: %s %d", value.Kind, value.UnsignedValue())
	}
}

func TestFixedIntegerConversionBuiltinRejectsOutOfRange(t *testing.T) {
	builtin := FixedIntegerConversionBuiltin(object.FixedU8)
	result := builtin.Fn(&object.Integer{Value: 256})
	errorObject, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected error, got %T", result)
	}
	if !strings.Contains(errorObject.Message, "outside u8 range") {
		t.Fatalf("unexpected error: %s", errorObject.Message)
	}
}

func TestFixedArithmeticBuiltins(t *testing.T) {
	max := object.NewFixedIntegerRaw(object.FixedU8, 255)
	one := object.NewFixedIntegerRaw(object.FixedU8, 1)

	wrapped := FixedArithmeticBuiltin(numeric.Wrapping, "+", "wrapAdd").Fn(max, one)
	if wrapped.(*object.FixedInteger).UnsignedValue() != 0 {
		t.Fatalf("expected wrapped result 0, got %s", wrapped.Inspect())
	}

	saturated := FixedArithmeticBuiltin(numeric.Saturating, "+", "satAdd").Fn(max, one)
	if saturated.(*object.FixedInteger).UnsignedValue() != 255 {
		t.Fatalf("expected saturated result 255, got %s", saturated.Inspect())
	}
}
