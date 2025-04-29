package builtins

import (
	"fmt"
	"zumbra/object"
)

func ShowBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}
			return nil
		},
	}
}
