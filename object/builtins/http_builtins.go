package builtins

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
var routeInvoker func(handler object.Object, args ...object.Object) (object.Object, error)

func SetRouteInvoker(invoker func(handler object.Object, args ...object.Object) (object.Object, error)) {
	routeInvoker = invoker
}

func CreateServerBuiltin() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			portObj, ok := args[0].(*object.Integer)
			if !ok {
				return NewError("argument to `server` must be INTEGER, got %s", args[0].Type())
			}

			mux := http.NewServeMux()
			for _, sr := range staticRoutes {
				mux.Handle(sr.RoutePrefix+"/", http.StripPrefix(sr.RoutePrefix, http.FileServer(http.Dir(sr.StaticDir))))
			}

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				route, params := matchRoute(r)
				if route == nil {
					http.NotFound(w, r)
					return
				}

				for _, mw := range route.Middlewares {
					if !mw(w, r) {
						return
					}
				}

				reqObj := buildRequestObject(r, params)
				resObj := object.NewHttpResponse()
				result, err := invokeRouteHandler(route.HandlerBody, reqObj, resObj)
				if err != nil {
					writeErrorResponse(w, http.StatusInternalServerError, err.Error())
					return
				}
				if errObj, ok := result.(*object.Error); ok {
					writeErrorResponse(w, http.StatusInternalServerError, errObj.Message)
					return
				}

				writeResponse(w, resObj, result)
			})

			portStr := fmt.Sprintf("%d", portObj.Value)
			srvr := &http.Server{Addr: ":" + portStr, Handler: mux}

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
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
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
			(&object.String{Value: "body"}).DictKey(): {Key: &object.String{Value: "body"}, Value: &object.String{Value: string(body)}},
		}}
	}}
}

func RegisterRoutesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 && len(args) != 3 {
			return NewError("wrong number of arguments. got=%d, want=2 or 3", len(args))
		}
		methodValue := "GET"
		pathIndex := 0
		handlerIndex := 1
		if len(args) == 3 {
			method, ok := args[0].(*object.String)
			if !ok {
				return NewError("method must be STRING")
			}
			methodValue = strings.ToUpper(method.Value)
			pathIndex = 1
			handlerIndex = 2
		}
		path, ok := args[pathIndex].(*object.String)
		if !ok {
			return NewError("path must be STRING")
		}
		registerRoutes = append(registerRoutes, Route{Method: methodValue, Path: path.Value, HandlerBody: args[handlerIndex], Middlewares: nil})
		return nil
	}}
}

func UseMiddlewaresBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		path, ok1 := args[0].(*object.String)
		middlewareName, ok2 := args[1].(*object.String)
		if !ok1 || !ok2 {
			return NewError("method and path must be STRING")
		}
		for i, route := range registerRoutes {
			if route.Path == path.Value && middlewareName.Value == "logger" {
				registerRoutes[i].Middlewares = append(registerRoutes[i].Middlewares, func(w http.ResponseWriter, r *http.Request) bool {
					fmt.Println("Request: ", r.Method, r.URL.Path)
					return true
				})
			}
		}
		return nil
	}}
}

func HtmlHandlerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("html(content) expects 1 argument")
		}
		str, ok := args[0].(*object.String)
		if !ok {
			return NewError("html(content) expects a STRING")
		}
		return &object.Builtin{Fn: func(_ ...object.Object) object.Object { return &object.String{Value: str.Value} }}
	}}
}

func ServerStaticBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		prefix, ok1 := args[0].(*object.String)
		dir, ok2 := args[1].(*object.String)
		if !ok1 || !ok2 {
			return NewError("method and path must be STRING")
		}
		staticRoutes = append(staticRoutes, StaticRoute{RoutePrefix: prefix.Value, StaticDir: dir.Value})
		return nil
	}}
}

func invokeRouteHandler(handler object.Object, req *object.HttpRequest, res *object.HttpResponse) (object.Object, error) {
	switch h := handler.(type) {
	case *object.String:
		return h, nil
	case *object.Builtin:
		return h.Fn(req, res), nil
	default:
		if routeInvoker == nil {
			return nil, fmt.Errorf("route handler invoker is not configured")
		}
		return routeInvoker(handler, req, res)
	}
}

func matchRoute(r *http.Request) (*Route, map[string]string) {
	for _, route := range registerRoutes {
		if route.Method != r.Method {
			continue
		}
		reqParts := splitPath(r.URL.Path)
		routeParts := splitPath(route.Path)
		if len(reqParts) != len(routeParts) {
			continue
		}
		params := map[string]string{}
		match := true
		for i := 0; i < len(reqParts); i++ {
			if strings.HasPrefix(routeParts[i], ":") {
				params[strings.TrimPrefix(routeParts[i], ":")] = reqParts[i]
				continue
			}
			if reqParts[i] != routeParts[i] {
				match = false
				break
			}
		}
		if match {
			routeCopy := route
			return &routeCopy, params
		}
	}
	return nil, nil
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func buildRequestObject(r *http.Request, params map[string]string) *object.HttpRequest {
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body.Close()
	var bodyObj object.Object = &object.Null{}
	if len(bodyBytes) > 0 {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(strings.ToLower(contentType), "application/json") {
			var decoded interface{}
			if err := json.Unmarshal(bodyBytes, &decoded); err == nil {
				bodyObj = goValueToObject(decoded)
			} else {
				bodyObj = &object.String{Value: string(bodyBytes)}
			}
		} else {
			bodyObj = &object.String{Value: string(bodyBytes)}
		}
	}
	return &object.HttpRequest{
		Method:  r.Method,
		Path:    r.URL.Path,
		Params:  mapToDict(params),
		Query:   valuesToDict(r.URL.Query()),
		Headers: headersToDict(r.Header),
		Body:    bodyObj,
		RawBody: string(bodyBytes),
	}
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"error":%q}`, message)))
}

func writeResponse(w http.ResponseWriter, res *object.HttpResponse, result object.Object) {
	for k, v := range res.Headers {
		w.Header().Set(k, v)
	}
	if res.StatusCode <= 0 {
		res.StatusCode = 200
	}
	payload := res.Body
	if !res.Written && payload == nil && result != nil && result.Type() != object.NULL_OBJ {
		payload = result
	}
	if res.ContentType != "" {
		w.Header().Set("Content-Type", res.ContentType)
	}
	if payload == nil {
		w.WriteHeader(res.StatusCode)
		return
	}
	if w.Header().Get("Content-Type") == "" {
		switch payload.Type() {
		case object.DICT_OBJ, object.ARRAY_OBJ:
			w.Header().Set("Content-Type", "application/json")
		default:
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
	}
	w.WriteHeader(res.StatusCode)
	_, _ = w.Write(renderPayload(payload, w.Header().Get("Content-Type")))
}

func renderPayload(obj object.Object, contentType string) []byte {
	if strings.Contains(contentType, "application/json") {
		b, err := json.Marshal(objectToGoValue(obj))
		if err == nil {
			return b
		}
	}
	switch v := obj.(type) {
	case *object.String:
		return []byte(v.Value)
	default:
		return []byte(v.Inspect())
	}
}

func mapToDict(input map[string]string) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for k, v := range input {
		key := &object.String{Value: k}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: v}}
	}
	return &object.Dict{Pairs: pairs}
}

func valuesToDict(values url.Values) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for k, arr := range values {
		key := &object.String{Value: k}
		if len(arr) == 1 {
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: arr[0]}}
		} else {
			elems := make([]object.Object, 0, len(arr))
			for _, item := range arr {
				elems = append(elems, &object.String{Value: item})
			}
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.Array{Elements: elems}}
		}
	}
	return &object.Dict{Pairs: pairs}
}

func headersToDict(headers http.Header) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for k, arr := range headers {
		key := &object.String{Value: strings.ToLower(k)}
		val := strings.Join(arr, ",")
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: val}}
	}
	return &object.Dict{Pairs: pairs}
}

func goValueToObject(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return &object.Null{}
	case string:
		return &object.String{Value: v}
	case bool:
		return &object.Boolean{Value: v}
	case float64:
		if float64(int64(v)) == v {
			return &object.Integer{Value: int64(v)}
		}
		return &object.Float{Value: v}
	case int:
		return &object.Integer{Value: int64(v)}
	case int64:
		return &object.Integer{Value: v}
	case []interface{}:
		elems := make([]object.Object, 0, len(v))
		for _, item := range v {
			elems = append(elems, goValueToObject(item))
		}
		return &object.Array{Elements: elems}
	case map[string]interface{}:
		pairs := map[object.DictKey]object.DictPair{}
		for keyStr, item := range v {
			key := &object.String{Value: keyStr}
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: goValueToObject(item)}
		}
		return &object.Dict{Pairs: pairs}
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}

func objectToGoValue(obj object.Object) interface{} {
	switch v := obj.(type) {
	case *object.Null:
		return nil
	case *object.String:
		return v.Value
	case *object.Boolean:
		return v.Value
	case *object.Integer:
		return v.Value
	case *object.Float:
		return v.Value
	case *object.Array:
		out := make([]interface{}, 0, len(v.Elements))
		for _, item := range v.Elements {
			out = append(out, objectToGoValue(item))
		}
		return out
	case *object.Dict:
		out := map[string]interface{}{}
		for _, pair := range v.Pairs {
			out[pair.Key.Inspect()] = objectToGoValue(pair.Value)
		}
		return out
	default:
		return v.Inspect()
	}
}

func ResponseMethod(res *object.HttpResponse, name string) object.Object {
	switch name {
	case "status":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("res.status(code) expects 1 argument")
			}
			code, ok := args[0].(*object.Integer)
			if !ok {
				return NewError("res.status(code) expects INTEGER")
			}
			res.StatusCode = int(code.Value)
			return res
		}}
	case "header":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("res.header(key, value) expects 2 arguments")
			}
			key, ok1 := args[0].(*object.String)
			value, ok2 := args[1].(*object.String)
			if !ok1 || !ok2 {
				return NewError("res.header(key, value) expects STRING arguments")
			}
			res.Headers[key.Value] = value.Value
			return res
		}}
	case "json":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("res.json(data) expects 1 argument")
			}
			res.ContentType = "application/json"
			res.Body = args[0]
			res.Written = true
			return res
		}}
	case "send":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("res.send(data) expects 1 argument")
			}
			res.Body = args[0]
			res.Written = true
			return res
		}}
	case "html":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("res.html(content) expects 1 argument")
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return NewError("res.html(content) expects STRING")
			}
			res.ContentType = "text/html; charset=utf-8"
			res.Body = str
			res.Written = true
			return res
		}}
	default:
		return nil
	}
}

func RequestAttr(req *object.HttpRequest, name string) object.Object {
	switch name {
	case "method":
		return &object.String{Value: req.Method}
	case "path":
		return &object.String{Value: req.Path}
	case "params":
		return req.Params
	case "query":
		return req.Query
	case "headers":
		return req.Headers
	case "body":
		return req.Body
	case "rawBody":
		return &object.String{Value: req.RawBody}
	default:
		return nil
	}
}

func parseNumericString(value string) object.Object {
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return &object.Integer{Value: i}
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return &object.Float{Value: f}
	}
	return &object.String{Value: value}
}
