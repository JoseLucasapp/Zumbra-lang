package builtins

import (
	"zumbra/collections"
	"zumbra/object"
)

func BytesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("bytes expects 1 argument, got %d", len(args))
		}
		value, err := collections.NewByteArray(args[0])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func ArrayOfBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("arrayOf expects 2 arguments, got %d", len(args))
		}
		value, err := collections.NewTypedArray(args[0], args[1])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func SliceBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("slice expects 3 arguments, got %d", len(args))
		}
		value, err := collections.NewSlice(args[0], args[1], args[2])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func FillBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("fill expects 2 arguments, got %d", len(args))
		}
		value, err := collections.Fill(args[0], args[1])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}
