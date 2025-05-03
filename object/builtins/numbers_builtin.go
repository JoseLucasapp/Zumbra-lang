package builtins

import (
	"math"
	"math/rand"
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

func GenerateRandomIntegerBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var x int = 10
			var y int = 0
			if len(args) != 2 {
				if len(args) == 1 {
					if args[0].Type() != object.INTEGER_OBJ {
						return NewError("argument to `generateRandomInteger` must be INTEGER, got %s", args[0].Type())
					}
					x = int(args[0].(*object.Integer).Value)
				}
			} else {
				if args[0].Type() != object.INTEGER_OBJ || args[1].Type() != object.INTEGER_OBJ {
					return NewError("first argument to `generateRandomInteger` must be INTEGER, got %s", args[0].Type())
				}
				x = int(args[1].(*object.Integer).Value)
				y = int(args[0].(*object.Integer).Value)
			}

			max := x
			min := y

			if max < min {
				max = y
				min = x
			}

			return NewInteger(int64(min) + int64(rand.Intn(max-min+1)))
		},
	}
}

func GenerateRandomFloatBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var x float64 = 10
			var y float64 = 0
			if len(args) != 2 {
				if len(args) == 1 {
					if args[0].Type() == object.FLOAT_OBJ {
						x = float64(args[0].(*object.Float).Value)
					}

					if args[0].Type() == object.INTEGER_OBJ {
						x = float64(args[0].(*object.Integer).Value)
					}

					if args[0].Type() != object.INTEGER_OBJ && args[0].Type() != object.FLOAT_OBJ {
						return NewError("All arguments to `generateRandomFloat` must be INT or FLOAT")
					}

				}
			} else {
				if (args[0].Type() != object.FLOAT_OBJ || args[1].Type() != object.FLOAT_OBJ) && (args[0].Type() != object.INTEGER_OBJ || args[1].Type() != object.INTEGER_OBJ) {
					return NewError("All arguments to `generateRandomFloat` must be FLOAT")
				}

				if args[0].Type() == object.FLOAT_OBJ {
					y = float64(args[0].(*object.Float).Value)
				}

				if args[0].Type() == object.INTEGER_OBJ {
					y = float64(args[0].(*object.Integer).Value)
				}

				if args[1].Type() == object.FLOAT_OBJ {
					x = float64(args[1].(*object.Float).Value)
				}

				if args[1].Type() == object.INTEGER_OBJ {
					x = float64(args[1].(*object.Integer).Value)
				}
			}

			max := x
			min := y

			if max < min {
				max = y
				min = x
			}

			return NewFloat(float64(min) + (rand.Float64() * (max - min)))
		},
	}
}
