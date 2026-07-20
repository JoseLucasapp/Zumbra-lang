package collections

import (
	"fmt"

	"zumbra/numeric"
	"zumbra/object"
)

func IntegerValue(value object.Object) (int64, error) {
	switch value := value.(type) {
	case *object.Integer:
		return value.Value, nil
	case *object.FixedInteger:
		if value.Kind.Signed() {
			return value.SignedValue(), nil
		}
		if value.UnsignedValue() > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("integer is too large for an index")
		}
		return int64(value.UnsignedValue()), nil
	default:
		return 0, fmt.Errorf("expected integer, got %s", value.Type())
	}
}

func NewByteArray(size object.Object) (object.Object, error) {
	length, err := IntegerValue(size)
	if err != nil {
		return nil, fmt.Errorf("bytes size: %w", err)
	}
	if length < 0 {
		return nil, fmt.Errorf("bytes size must be non-negative, got %d", length)
	}
	return &object.ByteArray{Data: make([]byte, int(length))}, nil
}

func NewTypedArray(kindValue, size object.Object) (object.Object, error) {
	kindName, ok := kindValue.(*object.String)
	if !ok {
		return nil, fmt.Errorf("arrayOf type must be string, got %s", kindValue.Type())
	}
	kind, ok := object.ParseFixedIntegerKind(kindName.Value)
	if !ok {
		return nil, fmt.Errorf("arrayOf type must be u8, u16, u32, u64, i8, i16, i32 or i64, got %q", kindName.Value)
	}
	length, err := IntegerValue(size)
	if err != nil {
		return nil, fmt.Errorf("arrayOf size: %w", err)
	}
	if length < 0 {
		return nil, fmt.Errorf("arrayOf size must be non-negative, got %d", length)
	}
	width := int(kind.Bits() / 8)
	return &object.TypedArray{Kind: kind, Data: make([]byte, int(length)*width), Length: int(length)}, nil
}

func Length(value object.Object) (int, bool) {
	switch value := value.(type) {
	case *object.Array:
		return len(value.Elements), true
	case *object.ByteArray:
		return len(value.Data), true
	case *object.TypedArray:
		return value.Length, true
	case *object.Slice:
		return value.Length, true
	case *object.String:
		return len(value.Value), true
	case *object.Dict:
		return len(value.Pairs), true
	default:
		return 0, false
	}
}

func Get(container, index object.Object) (object.Object, bool, error) {
	i, err := IntegerValue(index)
	if err != nil {
		switch container.Type() {
		case object.BYTE_ARRAY_OBJ, object.TYPED_ARRAY_OBJ, object.SLICE_OBJ:
			return nil, true, err
		default:
			return nil, false, nil
		}
	}

	switch value := container.(type) {
	case *object.Array:
		if i < 0 || i >= int64(len(value.Elements)) {
			return nil, true, fmt.Errorf("array index out of bounds: %d (length %d)", i, len(value.Elements))
		}
		return value.Elements[i], true, nil
	case *object.ByteArray:
		if i < 0 || i >= int64(len(value.Data)) {
			return nil, true, fmt.Errorf("byte array index out of bounds: %d (length %d)", i, len(value.Data))
		}
		return object.NewFixedIntegerRaw(object.FixedU8, uint64(value.Data[i])), true, nil
	case *object.TypedArray:
		if i < 0 || i >= int64(value.Length) {
			return nil, true, fmt.Errorf("typed array index out of bounds: %d (length %d)", i, value.Length)
		}
		return value.Read(int(i)), true, nil
	case *object.Slice:
		if i < 0 || i >= int64(value.Length) {
			return nil, true, fmt.Errorf("slice index out of bounds: %d (length %d)", i, value.Length)
		}
		return Get(value.Source, &object.Integer{Value: int64(value.Start) + i})
	default:
		return nil, false, nil
	}
}

func Set(container, index, value object.Object) (bool, error) {
	i, err := IntegerValue(index)
	if err != nil {
		switch container.Type() {
		case object.ARRAY_OBJ, object.BYTE_ARRAY_OBJ, object.TYPED_ARRAY_OBJ, object.SLICE_OBJ:
			return true, err
		default:
			return false, nil
		}
	}

	switch target := container.(type) {
	case *object.Array:
		if i < 0 || i >= int64(len(target.Elements)) {
			return true, fmt.Errorf("array index out of bounds: %d (length %d)", i, len(target.Elements))
		}
		target.Elements[i] = value
		return true, nil
	case *object.ByteArray:
		if i < 0 || i >= int64(len(target.Data)) {
			return true, fmt.Errorf("byte array index out of bounds: %d (length %d)", i, len(target.Data))
		}
		converted, err := numeric.Convert(object.FixedU8, value)
		if err != nil {
			return true, err
		}
		target.Data[i] = byte(converted.UnsignedValue())
		return true, nil
	case *object.TypedArray:
		if i < 0 || i >= int64(target.Length) {
			return true, fmt.Errorf("typed array index out of bounds: %d (length %d)", i, target.Length)
		}
		converted, err := numeric.Convert(target.Kind, value)
		if err != nil {
			return true, err
		}
		target.WriteRaw(int(i), converted.UnsignedValue())
		return true, nil
	case *object.Slice:
		if i < 0 || i >= int64(target.Length) {
			return true, fmt.Errorf("slice index out of bounds: %d (length %d)", i, target.Length)
		}
		return Set(target.Source, &object.Integer{Value: int64(target.Start) + i}, value)
	default:
		return false, nil
	}
}

func NewSlice(container, startValue, endValue object.Object) (object.Object, error) {
	length, ok := Length(container)
	if !ok || container.Type() == object.STRING_OBJ || container.Type() == object.DICT_OBJ {
		return nil, fmt.Errorf("slice expects array, byte array, typed array or slice, got %s", container.Type())
	}
	start, err := IntegerValue(startValue)
	if err != nil {
		return nil, fmt.Errorf("slice start: %w", err)
	}
	end, err := IntegerValue(endValue)
	if err != nil {
		return nil, fmt.Errorf("slice end: %w", err)
	}
	if start < 0 || end < start || end > int64(length) {
		return nil, fmt.Errorf("invalid slice range [%d:%d] for length %d", start, end, length)
	}
	if parent, ok := container.(*object.Slice); ok {
		return &object.Slice{Source: parent.Source, Start: parent.Start + int(start), Length: int(end - start)}, nil
	}
	return &object.Slice{Source: container, Start: int(start), Length: int(end - start)}, nil
}

func Fill(container, value object.Object) (object.Object, error) {
	length, ok := Length(container)
	if !ok || container.Type() == object.STRING_OBJ || container.Type() == object.DICT_OBJ {
		return nil, fmt.Errorf("fill expects array, byte array, typed array or slice, got %s", container.Type())
	}
	for i := 0; i < length; i++ {
		if _, err := Set(container, &object.Integer{Value: int64(i)}, value); err != nil {
			return nil, err
		}
	}
	return container, nil
}
