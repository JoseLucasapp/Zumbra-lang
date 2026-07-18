package builtins

import (
	"zumbra/numeric"
	"zumbra/object"
)

func FixedIntegerConversionBuiltin(kind object.FixedIntegerKind) *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("%s expects 1 argument, got %d", kind, len(args))
			}
			value, err := numeric.Convert(kind, args[0])
			if err != nil {
				return NewError("%s", err)
			}
			return value
		},
	}
}

func FixedArithmeticBuiltin(mode numeric.ArithmeticMode, operator string, name string) *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("%s expects 2 arguments, got %d", name, len(args))
			}
			value, err := numeric.Arithmetic(mode, operator, args[0], args[1])
			if err != nil {
				return NewError("%s", err)
			}
			return value
		},
	}
}
