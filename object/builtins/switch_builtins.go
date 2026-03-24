package builtins

import "zumbra/object"

func SwitchCaseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("wrong number of arguments. got=%d, want=3", len(args))
		}

		key, ok := args[0].(object.Dictable)
		if !ok {
			return NewError("argument 1 to `switchCase` must be STRING, INTEGER or BOOLEAN, got %s", args[0].Type())
		}

		casesObj, ok := args[1].(*object.Dict)
		if !ok {
			return NewError("argument 2 to `switchCase` must be DICT, got %s", args[1].Type())
		}

		if pair, exists := casesObj.Pairs[key.DictKey()]; exists {
			return pair.Value
		}

		return args[2]
	}}
}