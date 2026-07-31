package object

import "testing"

func TestTypedArrayStorageUsesExactElementWidth(t *testing.T) {
	values := &TypedArray{Kind: FixedU32, Data: make([]byte, 8), Length: 2}
	values.WriteRaw(1, 0xDEADBEEF)
	got := values.Read(1)
	if got.Kind != FixedU32 || got.UnsignedValue() != 0xDEADBEEF {
		t.Fatalf("unexpected typed value: %s/%x", got.Kind, got.UnsignedValue())
	}
}
