package collections

import (
	"testing"
	"zumbra/object"
)

func TestByteArrayReadWriteAndFill(t *testing.T) {
	memoryObject, err := NewByteArray(&object.Integer{Value: 4})
	if err != nil {
		t.Fatal(err)
	}
	memory := memoryObject.(*object.ByteArray)
	if _, err := Set(memory, &object.Integer{Value: 1}, &object.Integer{Value: 0xA9}); err != nil {
		t.Fatal(err)
	}
	value, handled, err := Get(memory, &object.Integer{Value: 1})
	if err != nil || !handled {
		t.Fatalf("get failed: handled=%v err=%v", handled, err)
	}
	fixed := value.(*object.FixedInteger)
	if fixed.Kind != object.FixedU8 || fixed.UnsignedValue() != 0xA9 {
		t.Fatalf("unexpected byte: %v", fixed)
	}
	if _, err := Fill(memory, object.NewFixedIntegerRaw(object.FixedU8, 0xFF)); err != nil {
		t.Fatal(err)
	}
	if memory.Data[0] != 0xFF || memory.Data[3] != 0xFF {
		t.Fatalf("fill failed: %v", memory.Data)
	}
}

func TestTypedArrayIsCompactAndPreservesKind(t *testing.T) {
	arrayObject, err := NewTypedArray(&object.String{Value: "u16"}, &object.Integer{Value: 3})
	if err != nil {
		t.Fatal(err)
	}
	array := arrayObject.(*object.TypedArray)
	if len(array.Data) != 6 {
		t.Fatalf("expected 6 bytes, got %d", len(array.Data))
	}
	if _, err := Set(array, &object.Integer{Value: 1}, &object.Integer{Value: 0xABCD}); err != nil {
		t.Fatal(err)
	}
	value, _, err := Get(array, &object.Integer{Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	fixed := value.(*object.FixedInteger)
	if fixed.Kind != object.FixedU16 || fixed.UnsignedValue() != 0xABCD {
		t.Fatalf("unexpected value: %s/%d", fixed.Kind, fixed.UnsignedValue())
	}
}

func TestSliceIsMutableView(t *testing.T) {
	memoryObject, _ := NewByteArray(&object.Integer{Value: 8})
	viewObject, err := NewSlice(memoryObject, &object.Integer{Value: 2}, &object.Integer{Value: 5})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Set(viewObject, &object.Integer{Value: 0}, &object.Integer{Value: 0x42}); err != nil {
		t.Fatal(err)
	}
	value, _, err := Get(memoryObject, &object.Integer{Value: 2})
	if err != nil {
		t.Fatal(err)
	}
	if value.(*object.FixedInteger).UnsignedValue() != 0x42 {
		t.Fatalf("slice did not update source")
	}
}

func TestTypedArrayRejectsOutOfRangeValue(t *testing.T) {
	arrayObject, _ := NewTypedArray(&object.String{Value: "u8"}, &object.Integer{Value: 1})
	if _, err := Set(arrayObject, &object.Integer{Value: 0}, &object.Integer{Value: 300}); err == nil {
		t.Fatal("expected range error")
	}
}
