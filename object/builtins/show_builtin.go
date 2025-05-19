package builtins

import (
	"fmt"
	"strings"
	"zumbra/object"
)

func ShowBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) == 0 {
				fmt.Println()
				return nil
			}

			if len(args) == 1 {
				fmt.Println(args[0].Inspect())
				return nil
			}

			formatObj, ok := args[0].(*object.String)

			if !ok {
				return NewError("First argument to `show` must be STRING, got %s", args[0].Type())
			}
			format := formatObj.Value
			values := []interface{}{}

			for _, arg := range args[1:] {
				values = append(values, arg.Inspect())
			}

			formatConverted := strings.ReplaceAll(format, "{}", "%v")

			fmt.Printf(formatConverted+"\n", values...)
			return nil

		},
	}
}
