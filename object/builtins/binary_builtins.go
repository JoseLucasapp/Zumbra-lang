package builtins

import (
	"zumbra/binarydata"
	"zumbra/object"
)

func ReadBytesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("readBytes expects 1 argument, got %d", len(args))
		}
		value, err := binarydata.ReadFile(args[0])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func WriteBytesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("writeBytes expects 2 arguments, got %d", len(args))
		}
		value, err := binarydata.WriteFile(args[0], args[1])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func ReadUnsignedBuiltin(width int, order binarydata.ByteOrder, name string) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("%s expects 2 arguments, got %d", name, len(args))
		}
		value, err := binarydata.ReadUnsigned(args[0], args[1], width, order)
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		return value
	}}
}

func WriteUnsignedBuiltin(width int, order binarydata.ByteOrder, name string) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("%s expects 3 arguments, got %d", name, len(args))
		}
		value, err := binarydata.WriteUnsigned(args[0], args[1], args[2], width, order)
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		return value
	}}
}

func CopyBytesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 5 {
			return NewError("copyBytes expects 5 arguments, got %d", len(args))
		}
		value, err := binarydata.Copy(args[0], args[1], args[2], args[3], args[4])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func BytesEqualBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("bytesEqual expects 2 arguments, got %d", len(args))
		}
		value, err := binarydata.Equal(args[0], args[1])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}

func SHA256Builtin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("sha256 expects 1 argument, got %d", len(args))
		}
		value, err := binarydata.SHA256(args[0])
		if err != nil {
			return NewError("%s", err)
		}
		return value
	}}
}
