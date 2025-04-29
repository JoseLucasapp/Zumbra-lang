package builtins

import (
	"fmt"
	"math"
	"strconv"
	"zumbra/object"
)

func ToStringParserBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			var value any

			switch obj := args[0].(type) {
			case *object.Integer:
				value = obj.Value
			case *object.Float:
				value = obj.Value
			case *object.Boolean:
				value = obj.Value
			default:
				return NewError("argument to `toString` not supported, got=%s", args[0].Type())
			}

			return NewString(fmt.Sprintf("%v", value))
		},
	}
}

func ToIntParserBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch obj := args[0].(type) {
			case *object.String:
				value, errors := strconv.Atoi(obj.Value)

				if errors != nil {
					return NewError("Error to parse string. %s", errors.Error())
				}

				return NewInteger(int64(value))
			case *object.Float:
				return NewInteger(int64(math.Floor(obj.Value)))
			case *object.Boolean:
				if obj.Value == true {
					return NewInteger(1)
				} else {
					return NewInteger(0)
				}
			case *object.Integer:
				return obj
			default:
				return NewError("argument to `toInt` not supported, got=%s", args[0].Type())
			}
		},
	}
}

func ToFloatParserBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch obj := args[0].(type) {
			case *object.String:
				value, errors := strconv.ParseFloat(obj.Value, 64)

				if errors != nil {
					return NewError("Error to parse string. %s", errors.Error())
				}

				return NewFloat(float64(value))
			case *object.Float:
				return obj
			case *object.Boolean:
				if obj.Value == true {
					return NewFloat(1)
				} else {
					return NewFloat(0)
				}
			case *object.Integer:
				return NewFloat(float64(obj.Value))
			default:
				return NewError("argument to `toFloat` not supported, got=%s", args[0].Type())
			}
		},
	}
}

func ToBoolParserBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch obj := args[0].(type) {
			case *object.String:
				return NewBoolean(obj.Value != "")
			case *object.Float:
				return NewBoolean(obj.Value != 0)
			case *object.Boolean:
				return obj
			case *object.Integer:
				return NewBoolean(obj.Value != 0)
			default:
				return NewError("argument to `toBool` not supported, got=%s", args[0].Type())
			}
		},
	}
}
