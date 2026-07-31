package builtins

import (
	"zumbra/object"
)

func SizeOfBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch arg := args[0].(type) {
			case *object.Array:
				return &object.Integer{Value: int64(len(arg.Elements))}
			case *object.ByteArray:
				return &object.Integer{Value: int64(len(arg.Data))}
			case *object.TypedArray:
				return &object.Integer{Value: int64(arg.Length)}
			case *object.Slice:
				return &object.Integer{Value: int64(arg.Length)}
			case *object.String:
				return &object.Integer{Value: int64(len(arg.Value))}
			case *object.Dict:
				return &object.Integer{Value: int64(len(arg.Pairs))}
			default:
				return NewError("argument to `sizeOf` not supported, got %s", args[0].Type())
			}
		},
	}
}
