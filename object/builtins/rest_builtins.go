package builtins

import "zumbra/object"

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
		if _, ok := args[0].(*object.String); !ok {
			return NewError("path must be STRING, got %s", args[0].Type())
		}
		return HTTPRouteBuiltin().Fn(legacyHTTPApp, &object.String{Value: method}, args[0], args[1])
	}}
}
