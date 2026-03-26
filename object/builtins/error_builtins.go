package builtins

import "zumbra/object"

var ErrorBuiltin = object.Builtin{
	Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return &object.Error{Message: "error() expects exactly 1 argument"}
		}

		return &object.Error{Message: args[0].Inspect()}
	},
}

var PanicBuiltin = object.Builtin{
	Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			panic("panic() expects exactly 1 argument")
		}

		panic(args[0].Inspect())
	},
}
