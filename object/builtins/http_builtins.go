package builtins

import (
	"fmt"
	"io"
	"net/http"
	"zumbra/object"
)

func CreateServerBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {

			if len(args) != 4 {
				return NewError("wrong number of arguments, server(3333, GET, '/', '<h1>Zumbra</h1>'). got=%d, want=4", len(args))
			}

			if args[0].Type() != object.INTEGER_OBJ {
				return NewError("First argument to `http` must be INTEGER, server(3333, GET, '/', '<h1>Zumbra</h1>'), got %s", args[0].Type())
			}

			if args[1].Type() != object.STRING_OBJ {
				return NewError("Second argument to `http` must be STRING, server(3333, GET, '/', '<h1>Zumbra</h1>'), got %s", args[1].Type())
			}

			if args[1].(*object.String).Value != "GET" {
				return NewError("Third argument to `http` must be GET, server(3333, GET, '/', '<h1>Zumbra</h1>'), got %s", args[2].Type())
			}

			if args[2].Type() != object.STRING_OBJ {
				return NewError("Third argument to `http` must be STRING, server(3333, GET, '/', '<h1>Zumbra</h1>'), got %s", args[2].Type())
			}

			if args[3].Type() != object.STRING_OBJ {
				return NewError("Fourth argument to `http` must be STRING, server(3333, GET, '/', '<h1>Zumbra</h1>'), got %s", args[3].Type())
			}

			serverPort := args[0].(*object.Integer).Value
			serverRoute := args[2].(*object.String).Value
			serverResponse := args[3].(*object.String).Value

			http.HandleFunc(serverRoute, func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(serverResponse))
			})

			if err := http.ListenAndServe(":"+fmt.Sprintf("%d", serverPort), nil); err != nil {
				return NewError("Failed to start server, server(3333, GET, '/', '<h1>Zumbra</h1>'). got %s", err)
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
