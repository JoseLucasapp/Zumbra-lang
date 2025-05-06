package builtins

import (
	"fmt"
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

func CapitalizeBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("argument to `capitalize` must be STRING, got %s", args[0].Type())
			}

			val := strings.Title(args[0].(*object.String).Value)

			return NewString(val)
		},
	}
}

func RemoveWhiteSpacesBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("argument to `removeWhiteSpaces` must be STRING, got %s", args[0].Type())
			}

			fmt.Println(args[0].(*object.String).Value)
			val := strings.ReplaceAll(args[0].(*object.String).Value, " ", "")

			fmt.Println(val)

			return NewString(val)
		},
	}
}

func ReplaceBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("argument to `replace` must be STRING, got %s", args[0].Type())
			}

			if args[1].Type() != object.STRING_OBJ {
				return NewError("argument to `replace` must be STRING, got %s", args[1].Type())
			}

			if args[2].Type() != object.STRING_OBJ {
				return NewError("argument to `replace` must be STRING, got %s", args[2].Type())
			}

			val := strings.ReplaceAll(args[0].(*object.String).Value, args[1].(*object.String).Value, args[2].(*object.String).Value)

			return NewString(val)
		},
	}
}
