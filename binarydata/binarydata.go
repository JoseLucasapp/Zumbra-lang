package binarydata

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"

	"zumbra/collections"
	"zumbra/numeric"
	"zumbra/object"
)

type ByteOrder string

const (
	LittleEndian ByteOrder = "little"
	BigEndian    ByteOrder = "big"
)

func ReadFile(path object.Object) (object.Object, error) {
	pathValue, ok := path.(*object.String)
	if !ok {
		return nil, fmt.Errorf("readBytes path must be string, got %s", path.Type())
	}
	data, err := os.ReadFile(pathValue.Value)
	if err != nil {
		return nil, fmt.Errorf("readBytes %q: %w", pathValue.Value, err)
	}
	return &object.ByteArray{Data: data}, nil
}

func WriteFile(path, buffer object.Object) (object.Object, error) {
	pathValue, ok := path.(*object.String)
	if !ok {
		return nil, fmt.Errorf("writeBytes path must be string, got %s", path.Type())
	}
	data, err := Bytes(buffer)
	if err != nil {
		return nil, fmt.Errorf("writeBytes buffer: %w", err)
	}
	if err := os.WriteFile(pathValue.Value, data, 0o644); err != nil {
		return nil, fmt.Errorf("writeBytes %q: %w", pathValue.Value, err)
	}
	return &object.Integer{Value: int64(len(data))}, nil
}

func ReadUnsigned(buffer, offset object.Object, width int, order ByteOrder) (object.Object, error) {
	data, err := Bytes(buffer)
	if err != nil {
		return nil, err
	}
	start, err := offsetValue(offset)
	if err != nil {
		return nil, err
	}
	if err := requireRange(start, width, len(data)); err != nil {
		return nil, err
	}
	window := data[start : start+width]
	var raw uint64
	switch width {
	case 2:
		if order == LittleEndian {
			raw = uint64(binary.LittleEndian.Uint16(window))
		} else {
			raw = uint64(binary.BigEndian.Uint16(window))
		}
	case 4:
		if order == LittleEndian {
			raw = uint64(binary.LittleEndian.Uint32(window))
		} else {
			raw = uint64(binary.BigEndian.Uint32(window))
		}
	case 8:
		if order == LittleEndian {
			raw = binary.LittleEndian.Uint64(window)
		} else {
			raw = binary.BigEndian.Uint64(window)
		}
	default:
		return nil, fmt.Errorf("unsupported integer width %d", width)
	}
	return object.NewFixedIntegerRaw(unsignedKind(width), raw), nil
}

func WriteUnsigned(buffer, offset, value object.Object, width int, order ByteOrder) (object.Object, error) {
	data, err := MutableBytes(buffer)
	if err != nil {
		return nil, err
	}
	start, err := offsetValue(offset)
	if err != nil {
		return nil, err
	}
	if err := requireRange(start, width, len(data)); err != nil {
		return nil, err
	}
	kind := unsignedKind(width)
	converted, err := numeric.Convert(kind, value)
	if err != nil {
		return nil, err
	}
	window := data[start : start+width]
	raw := converted.UnsignedValue()
	switch width {
	case 2:
		if order == LittleEndian {
			binary.LittleEndian.PutUint16(window, uint16(raw))
		} else {
			binary.BigEndian.PutUint16(window, uint16(raw))
		}
	case 4:
		if order == LittleEndian {
			binary.LittleEndian.PutUint32(window, uint32(raw))
		} else {
			binary.BigEndian.PutUint32(window, uint32(raw))
		}
	case 8:
		if order == LittleEndian {
			binary.LittleEndian.PutUint64(window, raw)
		} else {
			binary.BigEndian.PutUint64(window, raw)
		}
	default:
		return nil, fmt.Errorf("unsupported integer width %d", width)
	}
	return buffer, nil
}

func Copy(destination, destinationStart, source, sourceStart, length object.Object) (object.Object, error) {
	destinationBytes, err := MutableBytes(destination)
	if err != nil {
		return nil, fmt.Errorf("copyBytes destination: %w", err)
	}
	sourceBytes, err := Bytes(source)
	if err != nil {
		return nil, fmt.Errorf("copyBytes source: %w", err)
	}
	dstStart, err := offsetValue(destinationStart)
	if err != nil {
		return nil, fmt.Errorf("copyBytes destination offset: %w", err)
	}
	srcStart, err := offsetValue(sourceStart)
	if err != nil {
		return nil, fmt.Errorf("copyBytes source offset: %w", err)
	}
	count, err := offsetValue(length)
	if err != nil {
		return nil, fmt.Errorf("copyBytes length: %w", err)
	}
	if err := requireRange(dstStart, count, len(destinationBytes)); err != nil {
		return nil, fmt.Errorf("copyBytes destination: %w", err)
	}
	if err := requireRange(srcStart, count, len(sourceBytes)); err != nil {
		return nil, fmt.Errorf("copyBytes source: %w", err)
	}
	copy(destinationBytes[dstStart:dstStart+count], sourceBytes[srcStart:srcStart+count])
	return destination, nil
}

func Equal(first, second object.Object) (object.Object, error) {
	firstBytes, err := Bytes(first)
	if err != nil {
		return nil, fmt.Errorf("bytesEqual first buffer: %w", err)
	}
	secondBytes, err := Bytes(second)
	if err != nil {
		return nil, fmt.Errorf("bytesEqual second buffer: %w", err)
	}
	return &object.Boolean{Value: bytes.Equal(firstBytes, secondBytes)}, nil
}

func SHA256(buffer object.Object) (object.Object, error) {
	data, err := Bytes(buffer)
	if err != nil {
		return nil, fmt.Errorf("sha256 buffer: %w", err)
	}
	sum := sha256.Sum256(data)
	return &object.String{Value: fmt.Sprintf("%x", sum)}, nil
}

// Bytes returns the bytes represented by a byte-compatible collection.
// The returned slice shares storage with the source when possible.
func Bytes(value object.Object) ([]byte, error) {
	switch value := value.(type) {
	case *object.ByteArray:
		return value.Data, nil
	case *object.TypedArray:
		if value.Kind != object.FixedU8 && value.Kind != object.FixedI8 {
			return nil, fmt.Errorf("expected byte-compatible array, got arrayOf(%s)", value.Kind)
		}
		return value.Data, nil
	case *object.Slice:
		data, err := Bytes(value.Source)
		if err != nil {
			return nil, err
		}
		if value.Start < 0 || value.Length < 0 || value.Start+value.Length > len(data) {
			return nil, fmt.Errorf("invalid byte slice range [%d:%d] for length %d", value.Start, value.Start+value.Length, len(data))
		}
		return data[value.Start : value.Start+value.Length], nil
	default:
		return nil, fmt.Errorf("expected bytes, arrayOf(\"u8\"), arrayOf(\"i8\") or byte slice, got %s", value.Type())
	}
}

func MutableBytes(value object.Object) ([]byte, error) {
	return Bytes(value)
}

func offsetValue(value object.Object) (int, error) {
	integer, err := collections.IntegerValue(value)
	if err != nil {
		return 0, fmt.Errorf("offset must be integer: %w", err)
	}
	if integer < 0 {
		return 0, fmt.Errorf("offset must be non-negative, got %d", integer)
	}
	maxInt := int64(^uint(0) >> 1)
	if integer > maxInt {
		return 0, fmt.Errorf("offset is too large: %d", integer)
	}
	return int(integer), nil
}

func requireRange(start, width, length int) error {
	if width < 0 || start < 0 || start > length || width > length-start {
		return fmt.Errorf("byte range [%d:%d] is outside buffer length %d", start, start+width, length)
	}
	return nil
}

func unsignedKind(width int) object.FixedIntegerKind {
	switch width {
	case 2:
		return object.FixedU16
	case 4:
		return object.FixedU32
	case 8:
		return object.FixedU64
	default:
		return ""
	}
}
