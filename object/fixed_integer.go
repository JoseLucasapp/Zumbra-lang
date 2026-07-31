package object

import (
	"fmt"
	"strconv"
)

type FixedIntegerKind string

const (
	FixedU8  FixedIntegerKind = "u8"
	FixedU16 FixedIntegerKind = "u16"
	FixedU32 FixedIntegerKind = "u32"
	FixedU64 FixedIntegerKind = "u64"
	FixedI8  FixedIntegerKind = "i8"
	FixedI16 FixedIntegerKind = "i16"
	FixedI32 FixedIntegerKind = "i32"
	FixedI64 FixedIntegerKind = "i64"
)

const (
	U8_OBJ  ObjectType = "U8"
	U16_OBJ ObjectType = "U16"
	U32_OBJ ObjectType = "U32"
	U64_OBJ ObjectType = "U64"
	I8_OBJ  ObjectType = "I8"
	I16_OBJ ObjectType = "I16"
	I32_OBJ ObjectType = "I32"
	I64_OBJ ObjectType = "I64"
)

type FixedInteger struct {
	Kind FixedIntegerKind
	Raw  uint64
}

func ParseFixedIntegerKind(value string) (FixedIntegerKind, bool) {
	switch FixedIntegerKind(value) {
	case FixedU8, FixedU16, FixedU32, FixedU64,
		FixedI8, FixedI16, FixedI32, FixedI64:
		return FixedIntegerKind(value), true
	default:
		return "", false
	}
}

func (k FixedIntegerKind) Bits() uint8 {
	switch k {
	case FixedU8, FixedI8:
		return 8
	case FixedU16, FixedI16:
		return 16
	case FixedU32, FixedI32:
		return 32
	case FixedU64, FixedI64:
		return 64
	default:
		return 0
	}
}

func (k FixedIntegerKind) Signed() bool {
	switch k {
	case FixedI8, FixedI16, FixedI32, FixedI64:
		return true
	default:
		return false
	}
}

func (k FixedIntegerKind) Mask() uint64 {
	bits := k.Bits()
	if bits == 0 {
		return 0
	}
	if bits == 64 {
		return ^uint64(0)
	}
	return (uint64(1) << bits) - 1
}

func (k FixedIntegerKind) ObjectType() ObjectType {
	switch k {
	case FixedU8:
		return U8_OBJ
	case FixedU16:
		return U16_OBJ
	case FixedU32:
		return U32_OBJ
	case FixedU64:
		return U64_OBJ
	case FixedI8:
		return I8_OBJ
	case FixedI16:
		return I16_OBJ
	case FixedI32:
		return I32_OBJ
	case FixedI64:
		return I64_OBJ
	default:
		return ObjectType("FIXED_INTEGER")
	}
}

func NewFixedIntegerRaw(kind FixedIntegerKind, raw uint64) *FixedInteger {
	return &FixedInteger{Kind: kind, Raw: raw & kind.Mask()}
}

func NewFixedIntegerFromInt64(kind FixedIntegerKind, value int64) (*FixedInteger, error) {
	if kind.Bits() == 0 {
		return nil, fmt.Errorf("unknown fixed integer type %q", kind)
	}

	if kind.Signed() {
		min, max := kind.SignedBounds()
		if value < min || value > max {
			return nil, fmt.Errorf("value %d is outside %s range [%d, %d]", value, kind, min, max)
		}
		return NewFixedIntegerRaw(kind, uint64(value)), nil
	}

	if value < 0 {
		return nil, fmt.Errorf("value %d cannot be converted to %s", value, kind)
	}
	return NewFixedIntegerFromUint64(kind, uint64(value))
}

func NewFixedIntegerFromUint64(kind FixedIntegerKind, value uint64) (*FixedInteger, error) {
	if kind.Bits() == 0 {
		return nil, fmt.Errorf("unknown fixed integer type %q", kind)
	}

	if kind.Signed() {
		_, max := kind.SignedBounds()
		if value > uint64(max) {
			return nil, fmt.Errorf("value %d is outside %s range", value, kind)
		}
		return NewFixedIntegerRaw(kind, value), nil
	}

	if value > kind.Mask() {
		return nil, fmt.Errorf("value %d is outside %s range [0, %d]", value, kind, kind.Mask())
	}
	return NewFixedIntegerRaw(kind, value), nil
}

func (k FixedIntegerKind) SignedBounds() (int64, int64) {
	bits := k.Bits()
	if bits == 0 || !k.Signed() {
		return 0, 0
	}
	if bits == 64 {
		return -1 << 63, 1<<63 - 1
	}
	max := int64((uint64(1) << (bits - 1)) - 1)
	return -max - 1, max
}

func (i *FixedInteger) Type() ObjectType {
	return i.Kind.ObjectType()
}

func (i *FixedInteger) Inspect() string {
	if i.Kind.Signed() {
		return strconv.FormatInt(i.SignedValue(), 10)
	}
	return strconv.FormatUint(i.UnsignedValue(), 10)
}

func (i *FixedInteger) UnsignedValue() uint64 {
	return i.Raw & i.Kind.Mask()
}

func (i *FixedInteger) SignedValue() int64 {
	raw := i.UnsignedValue()
	if !i.Kind.Signed() {
		return int64(raw)
	}
	bits := i.Kind.Bits()
	if bits == 64 {
		return int64(raw)
	}
	if bits == 0 {
		return 0
	}
	signBit := uint64(1) << (bits - 1)
	if raw&signBit == 0 {
		return int64(raw)
	}
	return int64(raw | ^i.Kind.Mask())
}

func (i *FixedInteger) DictKey() DictKey {
	return DictKey{Type: i.Type(), Value: i.UnsignedValue()}
}

func IsFixedIntegerObject(value Object) bool {
	_, ok := value.(*FixedInteger)
	return ok
}
