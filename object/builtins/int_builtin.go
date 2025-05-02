package builtins

import (
	"math"
	"zumbra/object"
)

func BhaskaraBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return NewError("wrong number of arguments. got=%d, want=3", len(args))
			}
			if args[0].Type() != object.INTEGER_OBJ ||
				args[1].Type() != object.INTEGER_OBJ ||
				args[2].Type() != object.INTEGER_OBJ {
				return NewError("All arguments to `bhaskara` must be INT")
			}

			a := float64(args[0].(*object.Integer).Value)
			b := float64(args[1].(*object.Integer).Value)
			c := float64(args[2].(*object.Integer).Value)

			var dicriminant float64 = (math.Pow(b, 2)) - ((4 * a) * c)

			if dicriminant < 0 {
				return &object.Null{}
			}

			if dicriminant == 0 {
				x := -b / (2 * a)
				return &object.Float{Value: x}
			}

			sqrtD := math.Sqrt(dicriminant)

			x1 := (-b + sqrtD) / (2 * a)
			x2 := (-b - sqrtD) / (2 * a)

			return &object.Array{
				Elements: []object.Object{
					&object.Float{Value: x1},
					&object.Float{Value: x2},
				},
			}

		},
	}
}
