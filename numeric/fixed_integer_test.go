package numeric

import (
	"testing"

	"zumbra/object"
)

func TestFixedIntegerBinaryWrapsLikeHardware(t *testing.T) {
	left := object.NewFixedIntegerRaw(object.FixedU8, 255)
	result, handled, err := Binary("+", left, &object.Integer{Value: 1})
	if err != nil || !handled {
		t.Fatalf("unexpected result: handled=%v err=%v", handled, err)
	}
	value := result.(*object.FixedInteger)
	if value.Kind != object.FixedU8 || value.UnsignedValue() != 0 {
		t.Fatalf("expected 0u8, got %s %d", value.Kind, value.UnsignedValue())
	}
}

func TestFixedIntegerArithmeticModes(t *testing.T) {
	max := object.NewFixedIntegerRaw(object.FixedU8, 255)
	one := object.NewFixedIntegerRaw(object.FixedU8, 1)

	wrapped, err := Arithmetic(Wrapping, "+", max, one)
	if err != nil || wrapped.(*object.FixedInteger).UnsignedValue() != 0 {
		t.Fatalf("unexpected wrapping result: %v %v", wrapped, err)
	}

	if _, err := Arithmetic(Checked, "+", max, one); err == nil {
		t.Fatal("expected checked overflow")
	}

	saturated, err := Arithmetic(Saturating, "+", max, one)
	if err != nil || saturated.(*object.FixedInteger).UnsignedValue() != 255 {
		t.Fatalf("unexpected saturating result: %v %v", saturated, err)
	}
}
