package builtins

import (
	"crypto/sha256"
	"zumbra/object"
)

func HashCodeBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			toBeHashed := args[0].(*object.String).Value
			hash := sha256.New()
			hash.Write([]byte(toBeHashed))

			hashInBytes := hash.Sum(nil)
			return &object.String{Value: string(hashInBytes)}
		},
	}
}
