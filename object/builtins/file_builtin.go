package builtins

import (
	"os"
	"zumbra/object"
)

func FileBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			strArg, ok := args[0].(*object.String)
			if !ok {
				return NewError("argument to `file` must be STRING, got %s", args[0].Type())
			}

			content, err := os.ReadFile(strArg.Value)
			if err != nil {
				return NewError("failed to read file: %s", err)
			}

			return &object.String{Value: string(content)}
		},
	}
}

func ServeFileBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("serveFile(path) expects 1 argument")
			}

			path, ok := args[0].(*object.String)
			if !ok {
				return NewError("serveFile(path) expects STRING")
			}

			content, err := os.ReadFile(path.Value)
			if err != nil {
				return NewError("failed to read file: %s", err.Error())
			}

			return &object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.String{Value: string(content)}
				},
			}
		},
	}
}
