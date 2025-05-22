package builtins

import (
	"fmt"
	"io"
	"net/http"
	"zumbra/object"
)

var registerRoutes = map[string]string{}

func CreateServerBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=4", len(args))
			}

			portObj, ok := args[0].(*object.Integer)
			if !ok {
				return NewError("argument to `server` must be INTEGER, got %s", args[0].Type())
			}

			for path, response := range registerRoutes {
				http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(response))
				})
			}

			portStr := fmt.Sprintf("%d", portObj.Value)
			if err := http.ListenAndServe(":"+portStr, nil); err != nil {
				return NewError("Failed to start server on port %s. got %s", portStr, err)
			}

			return nil
		},
	}
}

func GetBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			if args[0].Type() != object.STRING_OBJ {
				return NewError("argument to `get` must be STRING, got %s", args[0].Type())
			}

			resp, err := http.Get(args[0].(*object.String).Value)
			if err != nil {
				return NewError("Failed to get, get('%s'). got %s", args[0].(*object.String).Value, err)
			}

			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return NewError("Failed to read body, get('%s'). got %s", args[0].(*object.String).Value, err)
			}

			return &object.Dict{Pairs: map[object.DictKey]object.DictPair{
				(&object.String{Value: "body"}).DictKey(): {
					Key:   &object.String{Value: "body"},
					Value: &object.String{Value: string(body)},
				},
			}}
		},
	}
}

func RegisterRoutesBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 2 {
				return NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			pathObj, ok1 := args[0].(*object.String)
			respObj, ok2 := args[1].(*object.String)

			if !ok1 || !ok2 {
				return NewError("argument to `registerRoutes` must be STRING, got %s", args[0].Type())
			}

			registerRoutes[pathObj.Value] = respObj.Value

			return nil
		},
	}
}
