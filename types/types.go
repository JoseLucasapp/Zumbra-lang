package types

type Kind string

const (
	Unknown Kind = "unknown"
	Int     Kind = "int"
	U8      Kind = "u8"
	U16     Kind = "u16"
	U32     Kind = "u32"
	U64     Kind = "u64"
	I8      Kind = "i8"
	I16     Kind = "i16"
	I32     Kind = "i32"
	I64     Kind = "i64"
	Float   Kind = "float"
	Bool    Kind = "bool"
	String  Kind = "string"
	Null    Kind = "null"
	Array   Kind = "array"
	Dict    Kind = "dict"
	Func    Kind = "function"
)

type Type struct {
	Kind   Kind
	Elem   *Type
	Key    *Type
	Value  *Type
	Params []*Type
	Return *Type
}

func Simple(kind Kind) *Type {
	return &Type{Kind: kind}
}

func ArrayOf(elem *Type) *Type {
	return &Type{
		Kind: Array,
		Elem: elem,
	}
}

func DictOf(key *Type, value *Type) *Type {
	return &Type{
		Kind:  Dict,
		Key:   key,
		Value: value,
	}
}

func FuncOf(params []*Type, ret *Type) *Type {
	return &Type{
		Kind:   Func,
		Params: params,
		Return: ret,
	}
}

func Same(a, b *Type) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Kind != b.Kind {
		return false
	}

	switch a.Kind {
	case Array:
		if a.Elem == nil || b.Elem == nil {
			return a.Elem == b.Elem
		}
		return Same(a.Elem, b.Elem)

	case Dict:
		if a.Key == nil || b.Key == nil || a.Value == nil || b.Value == nil {
			return a.Key == b.Key && a.Value == b.Value
		}
		return Same(a.Key, b.Key) && Same(a.Value, b.Value)

	case Func:
		if len(a.Params) != len(b.Params) {
			return false
		}
		for i := range a.Params {
			if !Same(a.Params[i], b.Params[i]) {
				return false
			}
		}
		if a.Return == nil || b.Return == nil {
			return a.Return == b.Return
		}
		return Same(a.Return, b.Return)
	}

	return true
}

func IsNumeric(t *Type) bool {
	if t == nil {
		return false
	}
	return IsInteger(t) || t.Kind == Float
}

func IsInteger(t *Type) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case Int, U8, U16, U32, U64, I8, I16, I32, I64:
		return true
	default:
		return false
	}
}

func IsFixedInteger(t *Type) bool {
	return IsInteger(t) && t.Kind != Int
}

func FixedIntegerKind(name string) (Kind, bool) {
	switch Kind(name) {
	case U8, U16, U32, U64, I8, I16, I32, I64:
		return Kind(name), true
	default:
		return Unknown, false
	}
}
