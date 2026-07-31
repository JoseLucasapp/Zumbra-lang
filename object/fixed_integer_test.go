package object

import "testing"

func TestFixedIntegerSignedAndUnsignedViews(t *testing.T) {
	unsigned := NewFixedIntegerRaw(FixedU8, 0xFF)
	if unsigned.UnsignedValue() != 255 || unsigned.Inspect() != "255" {
		t.Fatalf("unexpected u8 value: raw=%d inspect=%s", unsigned.UnsignedValue(), unsigned.Inspect())
	}

	signed := NewFixedIntegerRaw(FixedI8, 0xFF)
	if signed.SignedValue() != -1 || signed.Inspect() != "-1" {
		t.Fatalf("unexpected i8 value: signed=%d inspect=%s", signed.SignedValue(), signed.Inspect())
	}
}

func TestFixedIntegerCheckedConstruction(t *testing.T) {
	if _, err := NewFixedIntegerFromInt64(FixedU8, 256); err == nil {
		t.Fatal("expected u8 range error")
	}
	if _, err := NewFixedIntegerFromInt64(FixedI8, -128); err != nil {
		t.Fatalf("expected -128 to fit i8: %v", err)
	}
}
