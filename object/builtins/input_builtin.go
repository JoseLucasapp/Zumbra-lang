package builtins

import (
	"fmt"
	"zumbra/object"
)

func InputBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var input string
			if len(args) > 0 {
				return &object.String{Value: fmt.Sprintf("%v", args[0])}
			}
			fmt.Scanln(&input)
			return &object.String{Value: input}
		},
	}
}
