package object

import (
	"fmt"
	"math"
	"strconv"
)

var Builtins = []struct {
	Name    string
	Builtin *Builtin
}{
	{
		"sizeOf",
		&Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				switch arg := args[0].(type) {
				case *Array:
					return &Integer{Value: int64(len(arg.Elements))}
				case *String:
					return &Integer{Value: int64(len(arg.Value))}
				default:
					return NewError("argument to `sizeOf` not supported, got %s", args[0].Type())
				}
			},
		},
	},
	{
		"show",
		&Builtin{
			Fn: func(args ...Object) Object {
				for _, arg := range args {
					fmt.Println(arg.Inspect())
				}
				return nil
			},
		},
	},
	{
		"input",
		&Builtin{
			Fn: func(args ...Object) Object {
				var input string
				if len(args) > 0 {
					return &String{Value: fmt.Sprintf("%v", args[0])}
				}
				fmt.Scanln(&input)
				return &String{Value: input}
			},
		},
	},
	{
		"first",
		&Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `first` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*Array)
				if len(arr.Elements) > 0 {
					return arr.Elements[0]
				}

				return nil
			},
		},
	},
	{
		"last",
		&Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `last` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*Array)
				length := len(arr.Elements)
				if length > 0 {
					return arr.Elements[length-1]
				}

				return nil
			},
		},
	},
	{
		"allButFirst",
		&Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `allButFirst` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*Array)
				length := len(arr.Elements)
				if length > 0 {
					newElements := make([]Object, length-1, length-1)
					copy(newElements, arr.Elements[1:])
					return &Array{Elements: newElements}
				}

				return nil
			},
		},
	},
	{
		"addToArray",
		&Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return NewError("wrong number of arguments. got=%d, want=2", len(args))
				}
				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `addToArray` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*Array)

				arr.Elements = append(arr.Elements, args[1])
				return arr
			},
		},
	},
	{
		"removeFromArray",
		&Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return NewError("wrong number of arguments. got=%d, want=2", len(args))
				}
				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `removeFromArray` must be ARRAY, got %s", args[0].Type())
				}
				if args[1].Type() != INTEGER_OBJ {
					return NewError("index argument to `removeFromArray` must be INTEGER, got %s", args[1].Type())
				}

				arr := args[0].(*Array)
				index := args[1].(*Integer).Value

				if index < 0 || int(index) >= len(arr.Elements) {
					return NewError("index out of bounds: %d", index)
				}

				arr.Elements = append(arr.Elements[:index], arr.Elements[index+1:]...)
				return arr
			},
		},
	},
	{
		"max", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}
				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `max` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*Array)
				if len(arr.Elements) == 0 {
					return nil
				}
				max := arr.Elements[0]
				for _, el := range arr.Elements[1:] {
					if math.Max(float64(max.(*Integer).Value), float64(el.(*Integer).Value)) == float64(el.(*Integer).Value) {
						max = el
					}
				}
				return max
			},
		},
	},
	{
		"min", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}
				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `min` must be ARRAY, got %s", args[0].Type())
				}

				arr := args[0].(*Array)
				if len(arr.Elements) == 0 {
					return nil
				}

				min := arr.Elements[0]
				for _, el := range arr.Elements[1:] {
					if math.Min(float64(min.(*Integer).Value), float64(el.(*Integer).Value)) == float64(el.(*Integer).Value) {
						min = el
					}
				}
				return min
			},
		},
	},
	{
		"indexOf", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return NewError("wrong number of arguments. got=%d, want=2", len(args))
				}
				if args[0].Type() != ARRAY_OBJ {
					return NewError("argument to `indexOf` must be ARRAY, got %s", args[0].Type())
				}
				if args[1].Type() != INTEGER_OBJ && args[1].Type() != STRING_OBJ {
					return NewError("index argument to `indexOf` must be INTEGER, got %s", args[1].Type())
				}

				var index any
				var typeOf string

				arr := args[0].(*Array)
				if args[1].Type() == INTEGER_OBJ {
					index = args[1].(*Integer).Value
					typeOf = INTEGER_OBJ
				}

				if args[1].Type() == STRING_OBJ {
					index = args[1].(*String).Value
					typeOf = STRING_OBJ
				}

				for i, el := range arr.Elements {
					if typeOf == INTEGER_OBJ {
						if el.(*Integer).Value == index.(int64) {
							return NewInteger(int64(i))
						}
					}

					if typeOf == STRING_OBJ {
						if el.(*String).Value == index.(string) {
							return NewInteger(int64(i))
						}
					}

				}
				return NewInteger(-1)
			},
		},
	},
	{
		"addToDict", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 3 {
					return NewError("wrong number of arguments. got=%d, want=3", len(args))
				}

				if args[0].Type() != DICT_OBJ {
					return NewError("argument to `addToDict` must be DICT, got %s", args[0].Type())
				}

				if _, ok := args[1].(Dictable); !ok {
					return NewError("key must be hashable (STRING, INTEGER, BOOLEAN), got %s", args[1].Type())
				}

				dict := args[0].(*Dict)
				keyObj := args[1]
				valueObj := args[2]

				dictKey := keyObj.(Dictable).DictKey()

				pair := DictPair{
					Key:   keyObj,
					Value: valueObj,
				}

				dict.Pairs[dictKey] = pair

				return nil
			},
		},
	},
	{
		"deleteFromDict", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 2 {
					return NewError("wrong number of arguments. got=%d, want=2", len(args))
				}

				if args[0].Type() != DICT_OBJ {
					return NewError("argument to `deleteFromDict` must be DICT, got %s", args[0].Type())
				}

				if _, ok := args[1].(Dictable); !ok {
					return NewError("key must be hashable (STRING, INTEGER, BOOLEAN), got %s", args[1].Type())
				}

				dict := args[0].(*Dict)
				keyObj := args[1]

				dictKey := keyObj.(Dictable).DictKey()

				delete(dict.Pairs, dictKey)

				return nil
			},
		},
	},
	{
		"toString", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				var value any

				switch obj := args[0].(type) {
				case *Integer:
					value = obj.Value
				case *Float:
					value = obj.Value
				case *Boolean:
					value = obj.Value
				default:
					return NewError("argument to `toString` not supported, got=%s", args[0].Type())
				}

				return NewString(fmt.Sprintf("%v", value))
			},
		},
	},
	{
		"toInt", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				switch obj := args[0].(type) {
				case *String:
					value, errors := strconv.Atoi(obj.Value)

					if errors != nil {
						return NewError("Error to parse string. %s", errors.Error())
					}

					return NewInteger(int64(value))
				case *Float:
					return NewInteger(int64(math.Floor(obj.Value)))
				case *Boolean:
					if obj.Value == true {
						return NewInteger(1)
					} else {
						return NewInteger(0)
					}
				case *Integer:
					return obj
				default:
					return NewError("argument to `toInt` not supported, got=%s", args[0].Type())
				}
			},
		},
	},
	{
		"toFloat", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				switch obj := args[0].(type) {
				case *String:
					value, errors := strconv.ParseFloat(obj.Value, 64)

					if errors != nil {
						return NewError("Error to parse string. %s", errors.Error())
					}

					return NewFloat(float64(value))
				case *Float:
					return obj
				case *Boolean:
					if obj.Value == true {
						return NewFloat(1)
					} else {
						return NewFloat(0)
					}
				case *Integer:
					return NewFloat(float64(obj.Value))
				default:
					return NewError("argument to `toFloat` not supported, got=%s", args[0].Type())
				}
			},
		},
	},
	{
		"toBool", &Builtin{
			Fn: func(args ...Object) Object {
				if len(args) != 1 {
					return NewError("wrong number of arguments. got=%d, want=1", len(args))
				}

				switch obj := args[0].(type) {
				case *String:
					return NewBoolean(obj.Value != "")
				case *Float:
					return NewBoolean(obj.Value != 0)
				case *Boolean:
					return obj
				case *Integer:
					return NewBoolean(obj.Value != 0)
				default:
					return NewError("argument to `toBool` not supported, got=%s", args[0].Type())
				}
			},
		},
	},
}

func NewBoolean(value bool) *Boolean {
	return &Boolean{Value: value}
}
func NewFloat(value float64) *Float {
	return &Float{Value: value}
}
func NewString(value string) *String {
	return &String{Value: value}
}

func NewInteger(value int64) *Integer {
	return &Integer{Value: value}
}
func NewError(format string, a ...interface{}) *Error {
	return &Error{Message: fmt.Sprintf(format, a...)}
}

func GetBuiltinByName(name string) *Builtin {
	for _, builtin := range Builtins {
		if builtin.Name == name {
			return builtin.Builtin
		}
	}
	return nil
}
