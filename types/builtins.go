package types

func builtinType(name string) (*Type, bool) {
	switch name {
	case "show":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Null)), true

	case "input":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(String)), true

	case "sizeOf":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Int)), true

	case "toInt":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Int)), true

	case "toFloat":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Float)), true

	case "toString":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(String)), true

	case "toBool":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(Bool)), true

	case "first":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	case "last":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	default:
		return nil, false
	}
}
