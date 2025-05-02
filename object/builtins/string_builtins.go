package builtins

import (
	"strings"
	"zumbra/object"
)

func UppercaseBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("argument to `toUppercase` must be STRING, got %s", args[0].Type())
			}

			val := strings.ToUpper(args[0].(*object.String).Value)

			return NewString(val)
		},
	}
}

func LowercaseBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("argument to `toLowercase` must be STRING, got %s", args[0].Type())
			}

			val := strings.ToLower(args[0].(*object.String).Value)

			return NewString(val)
		},
	}
}
