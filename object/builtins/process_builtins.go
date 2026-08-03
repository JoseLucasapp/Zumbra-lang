package builtins

import (
	"os"
	"time"

	"zumbra/object"
)

func ProcessArgsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("processArgs expects 0 arguments, got %d", len(args))
		}
		values := make([]object.Object, len(os.Args))
		for index, value := range os.Args {
			values[index] = NewString(value)
		}
		return &object.Array{Elements: values}
	}}
}

func UnixTimeSecondsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("unixTimeSeconds expects 0 arguments, got %d", len(args))
		}
		return &object.FixedInteger{Kind: object.FixedU64, Raw: uint64(time.Now().Unix())}
	}}
}
