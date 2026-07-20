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

	case "u8":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U8)), true
	case "u16":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U16)), true
	case "u32":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U32)), true
	case "u64":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(U64)), true
	case "i8":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I8)), true
	case "i16":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I16)), true
	case "i32":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I32)), true
	case "i64":
		return FuncOf([]*Type{Simple(Unknown)}, Simple(I64)), true

	case "wrapAdd", "wrapSub", "wrapMul",
		"checkedAdd", "checkedSub", "checkedMul",
		"satAdd", "satSub", "satMul":
		return FuncOf([]*Type{Simple(Unknown), Simple(Unknown)}, Simple(Unknown)), true

	case "bytes":
		return FuncOf([]*Type{Simple(Int)}, ByteArrayOf()), true

	case "arrayOf":
		return FuncOf([]*Type{Simple(String), Simple(Int)}, TypedArrayOf(Simple(Unknown))), true

	case "slice":
		return FuncOf([]*Type{Simple(Unknown), Simple(Int), Simple(Int)}, SliceOf(Simple(Unknown))), true

	case "fill":
		return FuncOf([]*Type{Simple(Unknown), Simple(Unknown)}, Simple(Unknown)), true

	case "first":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	case "last":
		return FuncOf([]*Type{ArrayOf(Simple(Unknown))}, Simple(Unknown)), true

	default:
		return nil, false
	}
}
