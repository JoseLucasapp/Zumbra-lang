package object

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	BYTE_ARRAY_OBJ  ObjectType = "BYTE_ARRAY"
	TYPED_ARRAY_OBJ ObjectType = "TYPED_ARRAY"
	SLICE_OBJ       ObjectType = "SLICE"
)

// ByteArray is a compact mutable sequence of bytes.
type ByteArray struct {
	Data []byte
}

func (b *ByteArray) Type() ObjectType { return BYTE_ARRAY_OBJ }
func (b *ByteArray) Inspect() string {
	parts := make([]string, len(b.Data))
	for i, value := range b.Data {
		parts[i] = fmt.Sprintf("%d", value)
	}
	return "bytes[" + strings.Join(parts, ", ") + "]"
}

// TypedArray stores fixed-width integers in a compact little-endian byte buffer.
type TypedArray struct {
	Kind   FixedIntegerKind
	Data   []byte
	Length int
}

func (a *TypedArray) Type() ObjectType { return TYPED_ARRAY_OBJ }
func (a *TypedArray) Inspect() string {
	var out bytes.Buffer
	out.WriteString("arrayOf(")
	out.WriteString(string(a.Kind))
	out.WriteString(")[")
	for i := 0; i < a.Length; i++ {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(a.Read(i).Inspect())
	}
	out.WriteString("]")
	return out.String()
}

func (a *TypedArray) ElementBytes() int { return int(a.Kind.Bits() / 8) }

func (a *TypedArray) Read(index int) *FixedInteger {
	width := a.ElementBytes()
	offset := index * width
	raw := readLittleEndian(a.Data[offset:offset+width], width)
	return NewFixedIntegerRaw(a.Kind, raw)
}

func (a *TypedArray) WriteRaw(index int, raw uint64) {
	width := a.ElementBytes()
	offset := index * width
	writeLittleEndian(a.Data[offset:offset+width], width, raw)
}

// Slice is a mutable view over an array-like object. Changes affect the source.
type Slice struct {
	Source Object
	Start  int
	Length int
}

func (s *Slice) Type() ObjectType { return SLICE_OBJ }
func (s *Slice) Inspect() string  { return fmt.Sprintf("slice(%d, %d)", s.Start, s.Start+s.Length) }

func readLittleEndian(data []byte, width int) uint64 {
	switch width {
	case 1:
		return uint64(data[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(data))
	case 4:
		return uint64(binary.LittleEndian.Uint32(data))
	case 8:
		return binary.LittleEndian.Uint64(data)
	default:
		return 0
	}
}

func writeLittleEndian(data []byte, width int, value uint64) {
	switch width {
	case 1:
		data[0] = byte(value)
	case 2:
		binary.LittleEndian.PutUint16(data, uint16(value))
	case 4:
		binary.LittleEndian.PutUint32(data, uint32(value))
	case 8:
		binary.LittleEndian.PutUint64(data, value)
	}
}
