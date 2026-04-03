package types

type Kind string

const (
	Unknown Kind = "unknown"
	Int     Kind = "int"
	Float   Kind = "float"
	Bool    Kind = "bool"
	String  Kind = "string"
	Null    Kind = "null"
	Array   Kind = "array"
	Func    Kind = "function"
)

type Type struct {
	Kind   Kind
	Elem   *Type
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
	return t.Kind == Int || t.Kind == Float
}
