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

func GetFromDictBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `getFromDict` must be DICT, got %s", args[0].Type())
			}

			if _, ok := args[1].(object.Dictable); !ok {
				return NewError("key must be hashable (STRING, INTEGER, BOOLEAN), got %s", args[1].Type())
			}

			dict := args[0].(*object.Dict)
			keyObj := args[1]

			dictKey := keyObj.(object.Dictable).DictKey()

			pair, ok := dict.Pairs[dictKey]
			if !ok {
				return nil
			}

			return pair.Value
		},
	}
}

func DictKeysBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `dictKeys` must be DICT, got %s", args[0].Type())
			}

			dict := args[0].(*object.Dict)

			var keys []object.Object
			for _, key := range dict.Pairs {
				keys = append(keys, key.Key)
			}

			if len(keys) == 0 {
				return &object.Array{Elements: []object.Object{}}
			}

			return &object.Array{Elements: keys}

		},
	}
}

func DictValuesBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.DICT_OBJ {
				return NewError("argument to `dictValues` must be DICT, got %s", args[0].Type())
			}

			dict := args[0].(*object.Dict)

			var values []object.Object
			for _, value := range dict.Pairs {
				values = append(values, value.Value)
			}

			if len(values) == 0 {
				return &object.Array{Elements: []object.Object{}}
			}

			return &object.Array{Elements: values}

		},
	}
}
