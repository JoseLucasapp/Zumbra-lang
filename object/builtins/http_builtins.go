package builtins

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"zumbra/object"
)

type Route struct {
	Method      string
	Path        string
	HandlerBody object.Object
	Middlewares []func(http.ResponseWriter, *http.Request) bool
}

type StaticRoute struct {
	RoutePrefix string
	StaticDir   string
}

var staticRoutes []StaticRoute

var registerRoutes []Route

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

			for _, sr := range staticRoutes {
				http.Handle(sr.RoutePrefix+"/", http.StripPrefix(sr.RoutePrefix, http.FileServer(http.Dir(sr.StaticDir))))
			}

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				route := matchRoute(r)

				if route == nil {
					http.NotFound(w, r)
					return
				}

				for _, mw := range route.Middlewares {
					if !mw(w, r) {
						return
					}
				}

				switch handler := route.HandlerBody.(type) {
				case *object.String:
					w.Write([]byte(handler.Value))
				case *object.Builtin:
					result := handler.Fn()
					if str, ok := result.(*object.String); ok {
						w.Write([]byte(str.Value))
					} else {
						w.Write([]byte("function did not return string"))
					}
				default:
					w.Write([]byte("unsupported handler type"))
				}
			})
			portStr := fmt.Sprintf("%d", portObj.Value)
			srvr := &http.Server{Addr: ":" + portStr, Handler: nil}

			ln, err := net.Listen("tcp", srvr.Addr)

			if err != nil {
				fmt.Printf("Failed to bind to port %s. got %s\n", portStr, err)
				return NewError("Failed to bind to port %s. got %s", portStr, err)
			}

			fmt.Printf("Zumbra server started on port %s\n", portStr)

			if err := srvr.Serve(ln); err != nil {
				fmt.Printf("Server stopped unexpectedly. got %s\n", err)
				return NewError("Server stopped unexpectedly. got %s", err)
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

			if len(args) != 3 {
				return NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			method, ok1 := args[0].(*object.String)

			path, ok2 := args[1].(*object.String)
			handler := args[2]

			if !ok1 || !ok2 {
				return NewError("method and path must be STRING")
			}

			registerRoutes = append(registerRoutes, Route{
				Method:      strings.ToUpper(method.Value),
				Path:        path.Value,
				HandlerBody: handler,
				Middlewares: nil,
			})

			return nil
		},
	}
}

func UseMiddlewaresBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			path, ok1 := args[0].(*object.String)
			middlewareName, ok2 := args[1].(*object.String)

			if !ok1 || !ok2 {
				return NewError("method and path must be STRING")
			}

			for i, route := range registerRoutes {
				if route.Path == path.Value {
					if middlewareName.Value == "logger" {
						registerRoutes[i].Middlewares = append(registerRoutes[i].Middlewares, func(w http.ResponseWriter, r *http.Request) bool {
							fmt.Println("Request: ", r.Method, r.URL.Path)
							return true
						})
					}
				}
			}

			return nil
		},
	}
}

func HtmlHandlerBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("html(content) expects 1 argument")
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return NewError("html(content) expects a STRING")
			}

			return &object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.String{Value: str.Value}
				},
			}
		},
	}
}

func ServerStaticBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			prefix, ok1 := args[0].(*object.String)
			dir, ok2 := args[1].(*object.String)

			if !ok1 || !ok2 {
				return NewError("method and path must be STRING")
			}

			staticRoutes = append(staticRoutes, StaticRoute{
				RoutePrefix: prefix.Value,
				StaticDir:   dir.Value,
			})

			return nil
		},
	}
}

func matchRoute(r *http.Request) *Route {
	for _, route := range registerRoutes {
		if route.Method != r.Method {
			continue
		}

		reqParts := strings.Split(r.URL.Path, "/")
		routeParts := strings.Split(route.Path, "/")

		if len(reqParts) != len(routeParts) {
			continue
		}

		match := true
		for i := 0; i < len(reqParts); i++ {
			if strings.HasPrefix(routeParts[i], ":") {
				continue
			}

			if reqParts[i] != routeParts[i] {
				match = false
				break
			}
		}

		if match {
			return &route
		}
	}
	return nil
}
