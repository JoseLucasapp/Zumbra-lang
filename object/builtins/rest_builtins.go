package builtins

import (
	"strings"
	"zumbra/object"
)

func RestGetBuiltin() *object.Builtin    { return methodRouteBuiltin("GET") }
func RestPostBuiltin() *object.Builtin   { return methodRouteBuiltin("POST") }
func RestPutBuiltin() *object.Builtin    { return methodRouteBuiltin("PUT") }
func RestDeleteBuiltin() *object.Builtin { return methodRouteBuiltin("DELETE") }
func RestPatchBuiltin() *object.Builtin  { return methodRouteBuiltin("PATCH") }

func methodRouteBuiltin(method string) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("wrong number of arguments. got=%d, want=2", len(args))
		}

		path, ok := args[0].(*object.String)
		if !ok {
			return NewError("path must be STRING, got %s", args[0].Type())
		}

		registerRoutes = append(registerRoutes, Route{
			Method:      strings.ToUpper(method),
			Path:        path.Value,
			HandlerBody: args[1],
			Middlewares: nil,
		})

		return nil
	}}
}
