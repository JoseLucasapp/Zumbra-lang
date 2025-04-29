package builtins

import (
	"zumbra/object"
)

func AddToDictBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `addToDict` must be DICT, got %s", args[0].Type())
			}

			if _, ok := args[1].(object.Dictable); !ok {
				return NewError("key must be hashable (STRING, INTEGER, BOOLEAN), got %s", args[1].Type())
			}

			dict := args[0].(*object.Dict)
			keyObj := args[1]
			valueObj := args[2]

			dictKey := keyObj.(object.Dictable).DictKey()

			pair := object.DictPair{
				Key:   keyObj,
				Value: valueObj,
			}

			dict.Pairs[dictKey] = pair

			return nil
		},
	}
}

func DeleteFromDictBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `deleteFromDict` must be DICT, got %s", args[0].Type())
			}

			if _, ok := args[1].(object.Dictable); !ok {
				return NewError("key must be hashable (STRING, INTEGER, BOOLEAN), got %s", args[1].Type())
			}

			dict := args[0].(*object.Dict)
			keyObj := args[1]

			dictKey := keyObj.(object.Dictable).DictKey()

			delete(dict.Pairs, dictKey)

			return nil
		},
	}
}
