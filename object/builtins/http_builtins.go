package builtins

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"zumbra/binarydata"
	"zumbra/object"
)

const (
	defaultHTTPBodyLimit = int64(8 * 1024 * 1024)
	maximumHTTPBodyLimit = int64(64 * 1024 * 1024)
)

var (
	routeInvoker   func(handler object.Object, args ...object.Object) (object.Object, error)
	routeInvokerMu sync.Mutex
	legacyHTTPApp  = object.NewHttpApp()
)

func SetRouteInvoker(invoker func(handler object.Object, args ...object.Object) (object.Object, error)) {
	routeInvokerMu.Lock()
	routeInvoker = invoker
	routeInvokerMu.Unlock()
}

func invokeHTTPHandler(handler object.Object, args ...object.Object) (object.Object, error) {
	if builtin, ok := handler.(*object.Builtin); ok {
		return builtin.Fn(args...), nil
	}
	routeInvokerMu.Lock()
	invoker := routeInvoker
	routeInvokerMu.Unlock()
	if invoker == nil {
		return nil, fmt.Errorf("HTTP handler invoker is not configured")
	}
	// VM globals are shared. Serialize handler entry to avoid races in the
	// interpreted backend. Native builds use independent pthread stacks.
	routeInvokerMu.Lock()
	defer routeInvokerMu.Unlock()
	return invoker(handler, args...)
}

func httpString(value object.Object, name string) (string, *object.Error) {
	text, ok := value.(*object.String)
	if !ok {
		return "", NewError("%s expects string, got %s", name, value.Type())
	}
	return text.Value, nil
}

func httpInt(value object.Object, name string) (int64, *object.Error) {
	integer, ok := value.(*object.Integer)
	if !ok {
		return 0, NewError("%s expects int, got %s", name, value.Type())
	}
	return integer.Value, nil
}

func httpBool(value object.Object, name string) (bool, *object.Error) {
	boolean, ok := value.(*object.Boolean)
	if !ok {
		return false, NewError("%s expects bool, got %s", name, value.Type())
	}
	return boolean.Value, nil
}

func httpAppArg(value object.Object, name string) (*object.HttpApp, *object.Error) {
	app, ok := value.(*object.HttpApp)
	if !ok {
		return nil, NewError("%s expects HttpApp, got %s", name, value.Type())
	}
	return app, nil
}

func httpServerArg(value object.Object, name string) (*object.HttpServer, *object.Error) {
	server, ok := value.(*object.HttpServer)
	if !ok {
		return nil, NewError("%s expects HttpServer, got %s", name, value.Type())
	}
	return server, nil
}

func httpResponseArg(value object.Object, name string) (*object.HttpResponse, *object.Error) {
	response, ok := value.(*object.HttpResponse)
	if !ok {
		return nil, NewError("%s expects HttpResponse, got %s", name, value.Type())
	}
	return response, nil
}

func httpClientResponseArg(value object.Object, name string) (*object.HttpClientResponse, *object.Error) {
	response, ok := value.(*object.HttpClientResponse)
	if !ok {
		return nil, NewError("%s expects HttpClientResponse, got %s", name, value.Type())
	}
	return response, nil
}

func httpDict(value object.Object, name string) (*object.Dict, *object.Error) {
	if _, ok := value.(*object.Null); ok {
		return &object.Dict{Pairs: map[object.DictKey]object.DictPair{}}, nil
	}
	dict, ok := value.(*object.Dict)
	if !ok {
		return nil, NewError("%s expects dict, got %s", name, value.Type())
	}
	return dict, nil
}

func httpDictString(dict *object.Dict, key string) (string, bool) {
	if dict == nil {
		return "", false
	}
	k := &object.String{Value: key}
	pair, ok := dict.Pairs[k.DictKey()]
	if !ok {
		return "", false
	}
	value, ok := pair.Value.(*object.String)
	if !ok {
		return "", false
	}
	return value.Value, true
}

func httpDictBool(dict *object.Dict, key string) (bool, bool) {
	if dict == nil {
		return false, false
	}
	k := &object.String{Value: key}
	pair, ok := dict.Pairs[k.DictKey()]
	if !ok {
		return false, false
	}
	value, ok := pair.Value.(*object.Boolean)
	if !ok {
		return false, false
	}
	return value.Value, true
}

func httpDictInt(dict *object.Dict, key string) (int64, bool) {
	if dict == nil {
		return 0, false
	}
	k := &object.String{Value: key}
	pair, ok := dict.Pairs[k.DictKey()]
	if !ok {
		return 0, false
	}
	value, ok := pair.Value.(*object.Integer)
	if !ok {
		return 0, false
	}
	return value.Value, true
}

func httpMapToDict(input map[string]string) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for keyText, valueText := range input {
		key := &object.String{Value: keyText}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: valueText}}
	}
	return &object.Dict{Pairs: pairs}
}

func httpHeaderToDict(headers http.Header) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for header, values := range headers {
		key := &object.String{Value: strings.ToLower(header)}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: strings.Join(values, ",")}}
	}
	return &object.Dict{Pairs: pairs}
}

func httpValuesToDict(values url.Values) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for name, values := range values {
		key := &object.String{Value: name}
		if len(values) == 1 {
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: values[0]}}
			continue
		}
		elements := make([]object.Object, 0, len(values))
		for _, value := range values {
			elements = append(elements, &object.String{Value: value})
		}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.Array{Elements: elements}}
	}
	return &object.Dict{Pairs: pairs}
}

func httpCookiesToDict(cookies []*http.Cookie) *object.Dict {
	pairs := map[object.DictKey]object.DictPair{}
	for _, cookie := range cookies {
		key := &object.String{Value: cookie.Name}
		pairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.String{Value: cookie.Value}}
	}
	return &object.Dict{Pairs: pairs}
}

func httpObjectToGo(value object.Object) any {
	switch current := value.(type) {
	case nil, *object.Null:
		return nil
	case *object.String:
		return current.Value
	case *object.Boolean:
		return current.Value
	case *object.Integer:
		return current.Value
	case *object.Float:
		return current.Value
	case *object.Array:
		result := make([]any, 0, len(current.Elements))
		for _, element := range current.Elements {
			result = append(result, httpObjectToGo(element))
		}
		return result
	case *object.Dict:
		result := map[string]any{}
		for _, pair := range current.Pairs {
			result[pair.Key.Inspect()] = httpObjectToGo(pair.Value)
		}
		return result
	default:
		return current.Inspect()
	}
}

func httpGoToObject(value any) object.Object {
	switch current := value.(type) {
	case nil:
		return &object.Null{}
	case string:
		return &object.String{Value: current}
	case bool:
		return &object.Boolean{Value: current}
	case float64:
		if float64(int64(current)) == current {
			return &object.Integer{Value: int64(current)}
		}
		return &object.Float{Value: current}
	case []any:
		elements := make([]object.Object, 0, len(current))
		for _, element := range current {
			elements = append(elements, httpGoToObject(element))
		}
		return &object.Array{Elements: elements}
	case map[string]any:
		pairs := map[object.DictKey]object.DictPair{}
		for name, element := range current {
			key := &object.String{Value: name}
			pairs[key.DictKey()] = object.DictPair{Key: key, Value: httpGoToObject(element)}
		}
		return &object.Dict{Pairs: pairs}
	default:
		return &object.String{Value: fmt.Sprint(current)}
	}
}

func httpMarshal(value object.Object) ([]byte, error) {
	return json.Marshal(httpObjectToGo(value))
}

func httpUnmarshal(data []byte) object.Object {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return &object.Error{Message: err.Error()}
	}
	return httpGoToObject(value)
}

func HTTPAppBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 0 {
			return NewError("httpApp expects 0 arguments, got %d", len(args))
		}
		return object.NewHttpApp()
	}}
}

func HTTPRouteBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("httpRoute expects 4 arguments, got %d", len(args))
		}
		app, errObj := httpAppArg(args[0], "httpRoute")
		if errObj != nil {
			return errObj
		}
		method, errObj := httpString(args[1], "httpRoute method")
		if errObj != nil {
			return errObj
		}
		pattern, errObj := httpString(args[2], "httpRoute pattern")
		if errObj != nil {
			return errObj
		}
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "" || !strings.HasPrefix(pattern, "/") {
			return NewError("httpRoute requires a method and an absolute route pattern")
		}
		app.Mu.Lock()
		app.Routes = append(app.Routes, object.HttpRoute{Method: method, Pattern: pattern, Handler: args[3]})
		app.Mu.Unlock()
		return app
	}}
}

func HTTPUseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("httpUse expects 2 arguments, got %d", len(args))
		}
		app, errObj := httpAppArg(args[0], "httpUse")
		if errObj != nil {
			return errObj
		}
		app.Mu.Lock()
		app.Middlewares = append(app.Middlewares, args[1])
		app.Mu.Unlock()
		return app
	}}
}

func HTTPStaticBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("httpStatic expects 3 arguments, got %d", len(args))
		}
		app, errObj := httpAppArg(args[0], "httpStatic")
		if errObj != nil {
			return errObj
		}
		prefix, errObj := httpString(args[1], "httpStatic prefix")
		if errObj != nil {
			return errObj
		}
		directory, errObj := httpString(args[2], "httpStatic directory")
		if errObj != nil {
			return errObj
		}
		absolute, err := filepath.Abs(directory)
		if err != nil {
			return NewError("httpStatic directory: %s", err)
		}
		app.Mu.Lock()
		app.StaticRoutes = append(app.StaticRoutes, object.HttpStaticRoute{Prefix: strings.TrimSuffix(prefix, "/"), Directory: absolute})
		app.Mu.Unlock()
		return app
	}}
}

func HTTPLimitBodyBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("httpLimitBody expects 2 arguments, got %d", len(args))
		}
		app, errObj := httpAppArg(args[0], "httpLimitBody")
		if errObj != nil {
			return errObj
		}
		limit, errObj := httpInt(args[1], "httpLimitBody")
		if errObj != nil {
			return errObj
		}
		if limit <= 0 || limit > maximumHTTPBodyLimit {
			return NewError("httpLimitBody must be between 1 and %d bytes", maximumHTTPBodyLimit)
		}
		app.Mu.Lock()
		app.MaxBodyBytes = limit
		app.Mu.Unlock()
		return app
	}}
}

func HTTPCompressionBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("httpCompression expects 2 arguments, got %d", len(args))
		}
		app, errObj := httpAppArg(args[0], "httpCompression")
		if errObj != nil {
			return errObj
		}
		enabled, errObj := httpBool(args[1], "httpCompression")
		if errObj != nil {
			return errObj
		}
		app.Mu.Lock()
		app.Compression = enabled
		app.Mu.Unlock()
		return app
	}}
}

func objectArrayStrings(value object.Object, name string) ([]string, *object.Error) {
	array, ok := value.(*object.Array)
	if !ok {
		return nil, NewError("%s expects array<string>, got %s", name, value.Type())
	}
	result := make([]string, 0, len(array.Elements))
	for _, element := range array.Elements {
		text, ok := element.(*object.String)
		if !ok {
			return nil, NewError("%s expects array<string>", name)
		}
		result = append(result, text.Value)
	}
	return result, nil
}

func HTTPCorsBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 6 {
			return NewError("httpCors expects 6 arguments, got %d", len(args))
		}
		app, errObj := httpAppArg(args[0], "httpCors")
		if errObj != nil {
			return errObj
		}
		origins, errObj := objectArrayStrings(args[1], "httpCors origins")
		if errObj != nil {
			return errObj
		}
		methods, errObj := objectArrayStrings(args[2], "httpCors methods")
		if errObj != nil {
			return errObj
		}
		headers, errObj := objectArrayStrings(args[3], "httpCors headers")
		if errObj != nil {
			return errObj
		}
		credentials, errObj := httpBool(args[4], "httpCors credentials")
		if errObj != nil {
			return errObj
		}
		maxAge, errObj := httpInt(args[5], "httpCors maxAge")
		if errObj != nil {
			return errObj
		}
		app.Mu.Lock()
		app.CORS = object.HttpCorsConfig{Enabled: true, Origins: origins, Methods: methods, Headers: headers, AllowCredentials: credentials, MaxAgeSeconds: int(maxAge)}
		app.Mu.Unlock()
		return app
	}}
}

func httpServeBuiltin(tlsEnabled bool) *object.Builtin {
	name := "httpServe"
	expected := 3
	if tlsEnabled {
		name = "httpServeTLS"
		expected = 5
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		app, errObj := httpAppArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		host, errObj := httpString(args[1], name+" host")
		if errObj != nil {
			return errObj
		}
		port, errObj := httpInt(args[2], name+" port")
		if errObj != nil {
			return errObj
		}
		if port < 0 || port > 65535 {
			return NewError("%s port must be between 0 and 65535", name)
		}
		listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.FormatInt(port, 10)))
		if err != nil {
			return NewError("%s: %s", name, err)
		}
		server := &object.HttpServer{Listener: listener, Done: make(chan struct{})}
		server.Server = &http.Server{
			Handler:           http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { handleHTTPRequest(app, writer, request) }),
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		server.Running.Store(true)
		go func() {
			defer close(server.Done)
			var serveErr error
			if tlsEnabled {
				cert, _ := httpString(args[3], name+" certificate")
				key, _ := httpString(args[4], name+" key")
				serveErr = server.Server.ServeTLS(listener, cert, key)
			} else {
				serveErr = server.Server.Serve(listener)
			}
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				server.Err.Store(serveErr)
			}
			server.Running.Store(false)
		}()
		return server
	}}
}

func HTTPServeBuiltin() *object.Builtin    { return httpServeBuiltin(false) }
func HTTPServeTLSBuiltin() *object.Builtin { return httpServeBuiltin(true) }

func HTTPShutdownBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("httpShutdown expects 2 arguments, got %d", len(args))
		}
		server, errObj := httpServerArg(args[0], "httpShutdown")
		if errObj != nil {
			return errObj
		}
		timeout, errObj := httpInt(args[1], "httpShutdown timeout")
		if errObj != nil {
			return errObj
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
		defer cancel()
		if err := server.Server.Shutdown(ctx); err != nil {
			return &object.Boolean{Value: false}
		}
		select {
		case <-server.Done:
		case <-ctx.Done():
			return &object.Boolean{Value: false}
		}
		return &object.Boolean{Value: true}
	}}
}

func HTTPServerPortBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("httpServerPort expects 1 argument, got %d", len(args))
		}
		server, errObj := httpServerArg(args[0], "httpServerPort")
		if errObj != nil {
			return errObj
		}
		_, portText, err := net.SplitHostPort(server.Listener.Addr().String())
		if err != nil {
			return NewError("httpServerPort: %s", err)
		}
		port, _ := strconv.ParseInt(portText, 10, 64)
		return &object.Integer{Value: port}
	}}
}

func HTTPServerAddressBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("httpServerAddress expects 1 argument, got %d", len(args))
		}
		server, errObj := httpServerArg(args[0], "httpServerAddress")
		if errObj != nil {
			return errObj
		}
		host, _, err := net.SplitHostPort(server.Listener.Addr().String())
		if err != nil {
			host = server.Listener.Addr().String()
		}
		return &object.String{Value: host}
	}}
}

func HTTPServerRunningBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("httpServerRunning expects 1 argument, got %d", len(args))
		}
		server, errObj := httpServerArg(args[0], "httpServerRunning")
		if errObj != nil {
			return errObj
		}
		return &object.Boolean{Value: server.Running.Load()}
	}}
}

func HTTPResponseBuiltin(kind string) *object.Builtin {
	expected := 2
	if kind == "redirect" {
		expected = 2
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("http%s expects %d arguments, got %d", strings.Title(kind), expected, len(args))
		}
		status, errObj := httpInt(args[0], "http response status")
		if errObj != nil {
			return errObj
		}
		if status < 100 || status > 599 {
			return NewError("HTTP status must be between 100 and 599")
		}
		response := object.NewHttpResponse()
		response.StatusCode = int(status)
		response.Written = true
		switch kind {
		case "json":
			response.ContentType = "application/json"
			response.Body = args[1]
		case "html":
			text, errObj := httpString(args[1], "httpHtml")
			if errObj != nil {
				return errObj
			}
			response.ContentType = "text/html; charset=utf-8"
			response.Body = &object.String{Value: text}
		case "redirect":
			location, errObj := httpString(args[1], "httpRedirect")
			if errObj != nil {
				return errObj
			}
			response.Headers["Location"] = []string{location}
			response.ContentType = "text/plain; charset=utf-8"
			response.Body = &object.String{Value: http.StatusText(int(status))}
		default:
			response.ContentType = "text/plain; charset=utf-8"
			response.Body = args[1]
		}
		return response
	}}
}

func HTTPFileResponseBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("httpFile expects 2 arguments, got %d", len(args))
		}
		status, errObj := httpInt(args[0], "httpFile status")
		if errObj != nil {
			return errObj
		}
		path, errObj := httpString(args[1], "httpFile path")
		if errObj != nil {
			return errObj
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return NewError("httpFile: %s", err)
		}
		response := object.NewHttpResponse()
		response.StatusCode = int(status)
		response.RawBody = data
		response.ContentType = mime.TypeByExtension(filepath.Ext(path))
		if response.ContentType == "" {
			response.ContentType = http.DetectContentType(data)
		}
		response.Headers["Content-Disposition"] = []string{fmt.Sprintf("inline; filename=%q", filepath.Base(path))}
		response.Written = true
		return response
	}}
}

func HTTPHeaderBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("httpHeader expects 3 arguments, got %d", len(args))
		}
		response, errObj := httpResponseArg(args[0], "httpHeader")
		if errObj != nil {
			return errObj
		}
		name, errObj := httpString(args[1], "httpHeader name")
		if errObj != nil {
			return errObj
		}
		value, errObj := httpString(args[2], "httpHeader value")
		if errObj != nil {
			return errObj
		}
		response.Headers[name] = append(response.Headers[name], value)
		return response
	}}
}

func HTTPCookieBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("httpCookie expects 4 arguments, got %d", len(args))
		}
		response, errObj := httpResponseArg(args[0], "httpCookie")
		if errObj != nil {
			return errObj
		}
		name, errObj := httpString(args[1], "httpCookie name")
		if errObj != nil {
			return errObj
		}
		value, errObj := httpString(args[2], "httpCookie value")
		if errObj != nil {
			return errObj
		}
		options, errObj := httpDict(args[3], "httpCookie options")
		if errObj != nil {
			return errObj
		}
		cookie := &http.Cookie{Name: name, Value: value, Path: "/"}
		if option, ok := httpDictString(options, "path"); ok {
			cookie.Path = option
		}
		if option, ok := httpDictString(options, "domain"); ok {
			cookie.Domain = option
		}
		if option, ok := httpDictInt(options, "maxAge"); ok {
			cookie.MaxAge = int(option)
		}
		if option, ok := httpDictBool(options, "secure"); ok {
			cookie.Secure = option
		}
		if option, ok := httpDictBool(options, "httpOnly"); ok {
			cookie.HttpOnly = option
		}
		if option, ok := httpDictString(options, "sameSite"); ok {
			switch strings.ToLower(option) {
			case "strict":
				cookie.SameSite = http.SameSiteStrictMode
			case "none":
				cookie.SameSite = http.SameSiteNoneMode
			default:
				cookie.SameSite = http.SameSiteLaxMode
			}
		}
		response.Headers["Set-Cookie"] = append(response.Headers["Set-Cookie"], cookie.String())
		return response
	}}
}

func HTTPStreamBuiltin(sse bool) *object.Builtin {
	name := "httpStream"
	expected := 3
	if sse {
		name = "httpSSE"
		expected = 2
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != expected {
			return NewError("%s expects %d arguments, got %d", name, expected, len(args))
		}
		status, errObj := httpInt(args[0], name+" status")
		if errObj != nil {
			return errObj
		}
		channelIndex := 1
		contentType := "text/event-stream"
		if !sse {
			contentType, errObj = httpString(args[1], name+" contentType")
			if errObj != nil {
				return errObj
			}
			channelIndex = 2
		}
		channel, ok := args[channelIndex].(*object.Channel)
		if !ok {
			return NewError("%s expects Channel as its final argument", name)
		}
		response := object.NewHttpResponse()
		response.StatusCode = int(status)
		response.Stream = &object.HttpStream{Channel: channel, ContentType: contentType, SSE: sse}
		response.ContentType = contentType
		response.Written = true
		response.Headers["Cache-Control"] = []string{"no-cache"}
		response.Headers["Connection"] = []string{"keep-alive"}
		return response
	}}
}

func SSEEventBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 4 {
			return NewError("sseEvent expects 4 arguments, got %d", len(args))
		}
		data, errObj := httpString(args[0], "sseEvent data")
		if errObj != nil {
			return errObj
		}
		event, _ := httpString(args[1], "sseEvent event")
		id, _ := httpString(args[2], "sseEvent id")
		retry, errObj := httpInt(args[3], "sseEvent retry")
		if errObj != nil {
			return errObj
		}
		var builder strings.Builder
		if event != "" {
			builder.WriteString("event: ")
			builder.WriteString(event)
			builder.WriteByte('\n')
		}
		if id != "" {
			builder.WriteString("id: ")
			builder.WriteString(id)
			builder.WriteByte('\n')
		}
		if retry > 0 {
			fmt.Fprintf(&builder, "retry: %d\n", retry)
		}
		for _, line := range strings.Split(data, "\n") {
			builder.WriteString("data: ")
			builder.WriteString(line)
			builder.WriteByte('\n')
		}
		builder.WriteByte('\n')
		return &object.String{Value: builder.String()}
	}}
}

func HTTPRequestBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 5 {
			return NewError("httpRequest expects 5 arguments, got %d", len(args))
		}
		method, errObj := httpString(args[0], "httpRequest method")
		if errObj != nil {
			return errObj
		}
		target, errObj := httpString(args[1], "httpRequest URL")
		if errObj != nil {
			return errObj
		}
		headers, errObj := httpDict(args[2], "httpRequest headers")
		if errObj != nil {
			return errObj
		}
		timeout, errObj := httpInt(args[4], "httpRequest timeout")
		if errObj != nil {
			return errObj
		}
		bodyData, contentType, errObj := httpRequestBody(args[3])
		if errObj != nil {
			return errObj
		}
		request, err := http.NewRequest(strings.ToUpper(method), target, bytes.NewReader(bodyData))
		if err != nil {
			return NewError("httpRequest: %s", err)
		}
		for _, pair := range headers.Pairs {
			request.Header.Add(pair.Key.Inspect(), pair.Value.Inspect())
		}
		if contentType != "" && request.Header.Get("Content-Type") == "" {
			request.Header.Set("Content-Type", contentType)
		}
		client := &http.Client{Timeout: time.Duration(timeout) * time.Millisecond}
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		}
		response, err := client.Do(request)
		if err != nil {
			return NewError("httpRequest: %s", err)
		}
		defer response.Body.Close()
		limited := io.LimitReader(response.Body, maximumHTTPBodyLimit+1)
		body, err := io.ReadAll(limited)
		if err != nil {
			return NewError("httpRequest read response: %s", err)
		}
		if int64(len(body)) > maximumHTTPBodyLimit {
			return NewError("httpRequest response exceeds the 64 MiB safety limit")
		}
		return &object.HttpClientResponse{
			StatusCode: response.StatusCode,
			Status:     response.Status,
			URL:        response.Request.URL.String(),
			Headers:    httpHeaderToDict(response.Header),
			Cookies:    httpCookiesToDict(response.Cookies()),
			Body:       body,
		}
	}}
}

func httpRequestBody(value object.Object) ([]byte, string, *object.Error) {
	switch current := value.(type) {
	case *object.Null:
		return nil, "", nil
	case *object.String:
		return []byte(current.Value), "text/plain; charset=utf-8", nil
	case *object.Dict, *object.Array:
		data, err := httpMarshal(value)
		if err != nil {
			return nil, "", NewError("httpRequest JSON body: %s", err)
		}
		return data, "application/json", nil
	default:
		data, err := binarydata.Bytes(value)
		if err != nil {
			return nil, "", NewError("httpRequest body expects null, string, dict, array or byte buffer")
		}
		return data, "application/octet-stream", nil
	}
}

func HTTPClientStatusBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("httpStatus expects 1 argument, got %d", len(args))
		}
		response, errObj := httpClientResponseArg(args[0], "httpStatus")
		if errObj != nil {
			return errObj
		}
		return &object.Integer{Value: int64(response.StatusCode)}
	}}
}

func HTTPClientBodyBuiltin(bytesMode bool) *object.Builtin {
	name := "httpBody"
	if bytesMode {
		name = "httpBodyBytes"
	}
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("%s expects 1 argument, got %d", name, len(args))
		}
		response, errObj := httpClientResponseArg(args[0], name)
		if errObj != nil {
			return errObj
		}
		if bytesMode {
			result := &object.ByteArray{Data: make([]byte, len(response.Body))}
			copy(result.Data, response.Body)
			return result
		}
		return &object.String{Value: string(response.Body)}
	}}
}

func HTTPClientJSONBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("httpBodyJSON expects 1 argument, got %d", len(args))
		}
		response, errObj := httpClientResponseArg(args[0], "httpBodyJSON")
		if errObj != nil {
			return errObj
		}
		return httpUnmarshal(response.Body)
	}}
}

func HTTPClientHeadersBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("httpHeaders expects 1 argument, got %d", len(args))
		}
		response, errObj := httpClientResponseArg(args[0], "httpHeaders")
		if errObj != nil {
			return errObj
		}
		return response.Headers
	}}
}

func JSONStringifyBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("jsonStringify expects 1 argument, got %d", len(args))
		}
		data, err := httpMarshal(args[0])
		if err != nil {
			return NewError("jsonStringify: %s", err)
		}
		return &object.String{Value: string(data)}
	}}
}

func JSONParseZ11Builtin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("jsonParse expects 1 argument, got %d", len(args))
		}
		text, errObj := httpString(args[0], "jsonParse")
		if errObj != nil {
			return errObj
		}
		return httpUnmarshal([]byte(text))
	}}
}

func JWTSignHS256Builtin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 3 {
			return NewError("jwtSignHS256 expects 3 arguments, got %d", len(args))
		}
		claims, errObj := httpDict(args[0], "jwtSignHS256 claims")
		if errObj != nil {
			return errObj
		}
		secret, errObj := httpString(args[1], "jwtSignHS256 secret")
		if errObj != nil {
			return errObj
		}
		expires, errObj := httpInt(args[2], "jwtSignHS256 expires")
		if errObj != nil {
			return errObj
		}
		claimsMap, ok := httpObjectToGo(claims).(map[string]any)
		if !ok {
			return NewError("jwtSignHS256 claims must be a dict")
		}
		if expires > 0 {
			claimsMap["exp"] = time.Now().Unix() + expires
		}
		headerData, _ := json.Marshal(map[string]any{"alg": "HS256", "typ": "JWT"})
		claimsData, err := json.Marshal(claimsMap)
		if err != nil {
			return NewError("jwtSignHS256: %s", err)
		}
		encoding := base64.RawURLEncoding
		signingInput := encoding.EncodeToString(headerData) + "." + encoding.EncodeToString(claimsData)
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(signingInput))
		token := signingInput + "." + encoding.EncodeToString(mac.Sum(nil))
		return &object.String{Value: token}
	}}
}

func JWTVerifyHS256Builtin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("jwtVerifyHS256 expects 2 arguments, got %d", len(args))
		}
		token, errObj := httpString(args[0], "jwtVerifyHS256 token")
		if errObj != nil {
			return errObj
		}
		secret, errObj := httpString(args[1], "jwtVerifyHS256 secret")
		if errObj != nil {
			return errObj
		}
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			return &object.Array{Elements: []object.Object{&object.Boolean{Value: false}, &object.Null{}}}
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
		expected := mac.Sum(nil)
		actual, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil || !hmac.Equal(expected, actual) {
			return &object.Array{Elements: []object.Object{&object.Boolean{Value: false}, &object.Null{}}}
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return &object.Array{Elements: []object.Object{&object.Boolean{Value: false}, &object.Null{}}}
		}
		claims := httpUnmarshal(payload)
		if dict, ok := claims.(*object.Dict); ok {
			if exp, exists := httpDictInt(dict, "exp"); exists && time.Now().Unix() >= exp {
				return &object.Array{Elements: []object.Object{&object.Boolean{Value: false}, &object.Null{}}}
			}
		}
		return &object.Array{Elements: []object.Object{&object.Boolean{Value: true}, claims}}
	}}
}

func matchHTTPRoute(method, path string, routes []object.HttpRoute) (*object.HttpRoute, map[string]string) {
	requested := splitHTTPPath(path)
	for index := range routes {
		route := &routes[index]
		if route.Method != method && route.Method != "*" {
			continue
		}
		pattern := splitHTTPPath(route.Pattern)
		params := map[string]string{}
		matched := true
		requestIndex := 0
		for patternIndex, segment := range pattern {
			if strings.HasPrefix(segment, "*") {
				params[strings.TrimPrefix(segment, "*")] = strings.Join(requested[requestIndex:], "/")
				requestIndex = len(requested)
				break
			}
			if requestIndex >= len(requested) {
				matched = false
				break
			}
			if strings.HasPrefix(segment, ":") {
				params[strings.TrimPrefix(segment, ":")] = requested[requestIndex]
			} else if segment != requested[requestIndex] {
				matched = false
				break
			}
			requestIndex++
			if patternIndex == len(pattern)-1 && requestIndex != len(requested) {
				matched = false
			}
		}
		if matched && requestIndex == len(requested) {
			return route, params
		}
	}
	return nil, nil
}

func splitHTTPPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func mergeHTTPResponseState(base, returned *object.HttpResponse) *object.HttpResponse {
	if returned == nil {
		return base
	}
	if base == nil || base == returned {
		return returned
	}
	for name, values := range base.Headers {
		returned.Headers[name] = append(append([]string(nil), values...), returned.Headers[name]...)
	}
	if returned.StatusCode == 0 {
		returned.StatusCode = base.StatusCode
	}
	return returned
}

func handleHTTPRequest(app *object.HttpApp, writer http.ResponseWriter, request *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			http.Error(writer, fmt.Sprintf("internal server error: %v", recovered), http.StatusInternalServerError)
		}
	}()
	app.Mu.RLock()
	routes := append([]object.HttpRoute(nil), app.Routes...)
	middlewares := append([]object.Object(nil), app.Middlewares...)
	staticRoutes := append([]object.HttpStaticRoute(nil), app.StaticRoutes...)
	bodyLimit := app.MaxBodyBytes
	compression := app.Compression
	cors := app.CORS
	app.Mu.RUnlock()
	if bodyLimit <= 0 {
		bodyLimit = defaultHTTPBodyLimit
	}

	applyCORS(writer, request, cors)
	if cors.Enabled && request.Method == http.MethodOptions {
		writer.WriteHeader(http.StatusNoContent)
		return
	}
	if serveHTTPStatic(writer, request, staticRoutes) {
		return
	}
	route, params := matchHTTPRoute(request.Method, request.URL.Path, routes)
	if route == nil {
		http.NotFound(writer, request)
		return
	}
	requestObject, err := buildHTTPRequest(writer, request, params, bodyLimit)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errHTTPBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeHTTPError(writer, status, err.Error())
		return
	}
	responseObject := object.NewHttpResponse()
	for _, middleware := range middlewares {
		result, invokeErr := invokeHTTPHandler(middleware, requestObject, responseObject)
		if invokeErr != nil {
			writeHTTPError(writer, http.StatusInternalServerError, invokeErr.Error())
			return
		}
		if errObj, ok := result.(*object.Error); ok {
			writeHTTPError(writer, http.StatusInternalServerError, errObj.Message)
			return
		}
		if response, ok := result.(*object.HttpResponse); ok {
			responseObject = mergeHTTPResponseState(responseObject, response)
			writeHTTPResponse(writer, request, responseObject, compression)
			return
		}
		if boolean, ok := result.(*object.Boolean); ok && !boolean.Value {
			writeHTTPResponse(writer, request, responseObject, compression)
			return
		}
	}
	result, invokeErr := invokeHTTPHandler(route.Handler, requestObject, responseObject)
	if invokeErr != nil {
		writeHTTPError(writer, http.StatusInternalServerError, invokeErr.Error())
		return
	}
	if requestObject.Hijacked.Load() {
		return
	}
	if errObj, ok := result.(*object.Error); ok {
		writeHTTPError(writer, http.StatusInternalServerError, errObj.Message)
		return
	}
	if response, ok := result.(*object.HttpResponse); ok {
		responseObject = mergeHTTPResponseState(responseObject, response)
	} else if result != nil && result.Type() != object.NULL_OBJ && !responseObject.Written {
		responseObject.Body = result
		responseObject.Written = true
	}
	writeHTTPResponse(writer, request, responseObject, compression)
}

var errHTTPBodyTooLarge = errors.New("HTTP request body exceeds configured limit")

func buildHTTPRequest(writer http.ResponseWriter, request *http.Request, params map[string]string, limit int64) (*object.HttpRequest, error) {
	reader := io.LimitReader(request.Body, limit+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errHTTPBodyTooLarge
	}
	request.Body.Close()
	request.Body = io.NopCloser(bytes.NewReader(body))

	bodyObject := object.Object(&object.Null{})
	form := &object.Dict{Pairs: map[object.DictKey]object.DictPair{}}
	files := &object.Dict{Pairs: map[object.DictKey]object.DictPair{}}
	contentType := request.Header.Get("Content-Type")
	mediaType, parameters, _ := mime.ParseMediaType(contentType)
	switch mediaType {
	case "application/json":
		if len(body) > 0 {
			decoded := httpUnmarshal(body)
			if errObj, ok := decoded.(*object.Error); ok {
				return nil, fmt.Errorf("invalid JSON body: %s", errObj.Message)
			}
			bodyObject = decoded
		}
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, fmt.Errorf("invalid form body: %w", err)
		}
		form = httpValuesToDict(values)
		bodyObject = form
	case "multipart/form-data":
		boundary := parameters["boundary"]
		if boundary == "" {
			return nil, errors.New("multipart request is missing a boundary")
		}
		parsedForm, parsedFiles, err := parseHTTPMultipart(body, boundary)
		if err != nil {
			return nil, err
		}
		form, files = parsedForm, parsedFiles
		bodyObject = form
	default:
		if len(body) > 0 {
			bodyObject = &object.String{Value: string(body)}
		}
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	remoteHost, _, splitErr := net.SplitHostPort(request.RemoteAddr)
	if splitErr != nil {
		remoteHost = request.RemoteAddr
	}
	return &object.HttpRequest{
		Method: request.Method, Scheme: scheme, Host: request.Host, Path: request.URL.Path,
		RemoteAddress: remoteHost, Params: httpMapToDict(params), Query: httpValuesToDict(request.URL.Query()),
		Headers: httpHeaderToDict(request.Header), Cookies: httpCookiesToDict(request.Cookies()), Form: form, Files: files,
		Body: bodyObject, RawBody: string(body), RawBytes: body, Writer: writer, Request: request,
	}, nil
}

func parseHTTPMultipart(data []byte, boundary string) (*object.Dict, *object.Dict, error) {
	values := map[string][]string{}
	filePairs := map[object.DictKey]object.DictPair{}
	reader := multipart.NewReader(bytes.NewReader(data), boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("invalid multipart body: %w", err)
		}
		content, err := io.ReadAll(part)
		if err != nil {
			return nil, nil, err
		}
		if part.FileName() == "" {
			values[part.FormName()] = append(values[part.FormName()], string(content))
			continue
		}
		file := &object.HttpUploadedFile{FieldName: part.FormName(), Filename: part.FileName(), ContentType: part.Header.Get("Content-Type"), Data: content}
		key := &object.String{Value: part.FormName()}
		if existing, ok := filePairs[key.DictKey()]; ok {
			if array, ok := existing.Value.(*object.Array); ok {
				array.Elements = append(array.Elements, file)
			} else {
				filePairs[key.DictKey()] = object.DictPair{Key: key, Value: &object.Array{Elements: []object.Object{existing.Value, file}}}
			}
		} else {
			filePairs[key.DictKey()] = object.DictPair{Key: key, Value: file}
		}
	}
	return httpValuesToDict(values), &object.Dict{Pairs: filePairs}, nil
}

func applyCORS(writer http.ResponseWriter, request *http.Request, config object.HttpCorsConfig) {
	if !config.Enabled {
		return
	}
	origin := request.Header.Get("Origin")
	allowed := ""
	for _, candidate := range config.Origins {
		if candidate == "*" || candidate == origin {
			allowed = candidate
			if candidate == "*" && config.AllowCredentials {
				allowed = origin
			}
			break
		}
	}
	if allowed != "" {
		writer.Header().Set("Access-Control-Allow-Origin", allowed)
		writer.Header().Add("Vary", "Origin")
	}
	if len(config.Methods) > 0 {
		writer.Header().Set("Access-Control-Allow-Methods", strings.Join(config.Methods, ", "))
	}
	if len(config.Headers) > 0 {
		writer.Header().Set("Access-Control-Allow-Headers", strings.Join(config.Headers, ", "))
	}
	if len(config.ExposeHeaders) > 0 {
		writer.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
	}
	if config.AllowCredentials {
		writer.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if config.MaxAgeSeconds > 0 {
		writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAgeSeconds))
	}
}

func serveHTTPStatic(writer http.ResponseWriter, request *http.Request, routes []object.HttpStaticRoute) bool {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		return false
	}
	for _, route := range routes {
		if request.URL.Path != route.Prefix && !strings.HasPrefix(request.URL.Path, route.Prefix+"/") {
			continue
		}
		relative := strings.TrimPrefix(request.URL.Path, route.Prefix)
		relative = strings.TrimPrefix(relative, "/")
		clean := filepath.Clean(relative)
		if clean == "." {
			clean = "index.html"
		}
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			http.Error(writer, "invalid static path", http.StatusBadRequest)
			return true
		}
		full := filepath.Join(route.Directory, clean)
		absolute, err := filepath.Abs(full)
		if err != nil || (absolute != route.Directory && !strings.HasPrefix(absolute, route.Directory+string(os.PathSeparator))) {
			http.Error(writer, "invalid static path", http.StatusBadRequest)
			return true
		}
		http.ServeFile(writer, request, absolute)
		return true
	}
	return false
}

func writeHTTPError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	data, _ := json.Marshal(map[string]string{"error": message})
	_, _ = writer.Write(data)
}

func writeHTTPResponse(writer http.ResponseWriter, request *http.Request, response *object.HttpResponse, compression bool) {
	if response == nil {
		response = object.NewHttpResponse()
	}
	for name, values := range response.Headers {
		for _, value := range values {
			writer.Header().Add(name, value)
		}
	}
	if response.ContentType != "" {
		writer.Header().Set("Content-Type", response.ContentType)
	}
	if response.Stream != nil {
		writeHTTPStream(writer, response)
		return
	}
	payload := response.RawBody
	if payload == nil && response.Body != nil {
		if strings.Contains(strings.ToLower(response.ContentType), "application/json") || response.Body.Type() == object.DICT_OBJ || response.Body.Type() == object.ARRAY_OBJ {
			marshaled, err := httpMarshal(response.Body)
			if err != nil {
				writeHTTPError(writer, http.StatusInternalServerError, err.Error())
				return
			}
			payload = marshaled
			if writer.Header().Get("Content-Type") == "" {
				writer.Header().Set("Content-Type", "application/json")
			}
		} else if bytesValue, err := binarydata.Bytes(response.Body); err == nil {
			payload = bytesValue
		} else {
			payload = []byte(response.Body.Inspect())
		}
	}
	if writer.Header().Get("Content-Type") == "" {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	if compression && len(payload) >= 512 && strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
		var compressed bytes.Buffer
		gzipWriter := gzip.NewWriter(&compressed)
		_, _ = gzipWriter.Write(payload)
		_ = gzipWriter.Close()
		payload = compressed.Bytes()
		writer.Header().Set("Content-Encoding", "gzip")
		writer.Header().Add("Vary", "Accept-Encoding")
	}
	status := response.StatusCode
	if status <= 0 {
		status = http.StatusOK
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	writer.WriteHeader(status)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(payload)
	}
}

func writeHTTPStream(writer http.ResponseWriter, response *object.HttpResponse) {
	stream := response.Stream
	if stream == nil || stream.Channel == nil {
		writeHTTPError(writer, http.StatusInternalServerError, "invalid HTTP stream")
		return
	}
	writer.Header().Set("Content-Type", stream.ContentType)
	writer.Header().Set("Transfer-Encoding", "chunked")
	status := response.StatusCode
	if status <= 0 {
		status = http.StatusOK
	}
	writer.WriteHeader(status)
	flusher, _ := writer.(http.Flusher)
	for {
		value, open := stream.Channel.Receive()
		if !open {
			break
		}
		data, errObj := httpStreamData(value)
		if errObj != nil {
			break
		}
		if len(data) == 0 {
			continue
		}
		if _, err := writer.Write(data); err != nil {
			break
		}
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func httpStreamData(value object.Object) ([]byte, *object.Error) {
	if text, ok := value.(*object.String); ok {
		return []byte(text.Value), nil
	}
	data, err := binarydata.Bytes(value)
	if err != nil {
		return nil, NewError("HTTP stream chunks must be string or byte-compatible buffers")
	}
	return data, nil
}

func prependHTTPReceiver(receiver object.Object, builtin *object.Builtin) *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		all := make([]object.Object, 0, len(args)+1)
		all = append(all, receiver)
		all = append(all, args...)
		return builtin.Fn(all...)
	}}
}

// AppMethod exposes the high-level API directly on HttpApp while keeping the
// underlying implementation in the same builtins used by the functional API.
func AppMethod(app *object.HttpApp, name string) object.Object {
	switch name {
	case "route":
		return prependHTTPReceiver(app, HTTPRouteBuiltin())
	case "get", "post", "put", "patch", "delete":
		method := strings.ToUpper(name)
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("app.%s expects 2 arguments", name)
			}
			return HTTPRouteBuiltin().Fn(app, &object.String{Value: method}, args[0], args[1])
		}}
	case "use":
		return prependHTTPReceiver(app, HTTPUseBuiltin())
	case "static":
		return prependHTTPReceiver(app, HTTPStaticBuiltin())
	case "bodyLimit":
		return prependHTTPReceiver(app, HTTPLimitBodyBuiltin())
	case "compression":
		return prependHTTPReceiver(app, HTTPCompressionBuiltin())
	case "cors":
		return prependHTTPReceiver(app, HTTPCorsBuiltin())
	case "listen":
		return prependHTTPReceiver(app, HTTPServeBuiltin())
	case "listenTls":
		return prependHTTPReceiver(app, HTTPServeTLSBuiltin())
	default:
		return nil
	}
}

func ServerMethod(server *object.HttpServer, name string) object.Object {
	switch name {
	case "shutdown":
		return prependHTTPReceiver(server, HTTPShutdownBuiltin())
	case "port":
		return prependHTTPReceiver(server, HTTPServerPortBuiltin())
	case "address":
		return prependHTTPReceiver(server, HTTPServerAddressBuiltin())
	case "running":
		return prependHTTPReceiver(server, HTTPServerRunningBuiltin())
	default:
		return nil
	}
}

// RequestAttr and ResponseMethod preserve the legacy attribute APIs while
// exposing the richer Z11 request/response fields.
func RequestAttr(request *object.HttpRequest, name string) object.Object {
	switch name {
	case "method":
		return &object.String{Value: request.Method}
	case "scheme":
		return &object.String{Value: request.Scheme}
	case "host":
		return &object.String{Value: request.Host}
	case "path":
		return &object.String{Value: request.Path}
	case "remoteAddress":
		return &object.String{Value: request.RemoteAddress}
	case "params":
		return request.Params
	case "query":
		return request.Query
	case "headers":
		return request.Headers
	case "cookies":
		return request.Cookies
	case "form":
		return request.Form
	case "files":
		return request.Files
	case "body":
		return request.Body
	case "rawBody":
		return &object.String{Value: request.RawBody}
	case "rawBytes":
		result := &object.ByteArray{Data: make([]byte, len(request.RawBytes))}
		copy(result.Data, request.RawBytes)
		return result
	default:
		return nil
	}
}

func ResponseMethod(response *object.HttpResponse, name string) object.Object {
	switch name {
	case "status":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("response.status expects 1 argument")
			}
			status, errObj := httpInt(args[0], "response.status")
			if errObj != nil {
				return errObj
			}
			response.StatusCode = int(status)
			return response
		}}
	case "header":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return NewError("response.header expects 2 arguments")
			}
			header, errObj := httpString(args[0], "response.header")
			if errObj != nil {
				return errObj
			}
			value, errObj := httpString(args[1], "response.header")
			if errObj != nil {
				return errObj
			}
			response.Headers[header] = append(response.Headers[header], value)
			return response
		}}
	case "json", "send", "html":
		return &object.Builtin{Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return NewError("response.%s expects 1 argument", name)
			}
			response.Body = args[0]
			response.Written = true
			if name == "json" {
				response.ContentType = "application/json"
			} else if name == "html" {
				response.ContentType = "text/html; charset=utf-8"
			}
			return response
		}}
	default:
		return nil
	}
}

func ClientResponseAttr(response *object.HttpClientResponse, name string) object.Object {
	switch name {
	case "statusCode":
		return &object.Integer{Value: int64(response.StatusCode)}
	case "status":
		return &object.String{Value: response.Status}
	case "url":
		return &object.String{Value: response.URL}
	case "headers":
		return response.Headers
	case "cookies":
		return response.Cookies
	case "body":
		return &object.String{Value: string(response.Body)}
	case "bytes":
		result := &object.ByteArray{Data: make([]byte, len(response.Body))}
		copy(result.Data, response.Body)
		return result
	default:
		return nil
	}
}

func HTTPFileAttr(file *object.HttpUploadedFile, name string) object.Object {
	switch name {
	case "fieldName":
		return &object.String{Value: file.FieldName}
	case "filename":
		return &object.String{Value: file.Filename}
	case "contentType":
		return &object.String{Value: file.ContentType}
	case "size":
		return &object.Integer{Value: int64(len(file.Data))}
	case "data":
		result := &object.ByteArray{Data: make([]byte, len(file.Data))}
		copy(result.Data, file.Data)
		return result
	default:
		return nil
	}
}

// Legacy compatibility -------------------------------------------------------

func CreateServerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("server expects 1 argument")
		}
		server := HTTPServeBuiltin().Fn(legacyHTTPApp, &object.String{Value: "0.0.0.0"}, args[0])
		if errObj, ok := server.(*object.Error); ok {
			return errObj
		}
		handle := server.(*object.HttpServer)
		<-handle.Done
		return &object.Null{}
	}}
}

func GetBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("get expects 1 argument")
		}
		response := HTTPRequestBuiltin().Fn(&object.String{Value: "GET"}, args[0], &object.Dict{Pairs: map[object.DictKey]object.DictPair{}}, &object.Null{}, &object.Integer{Value: 30000})
		if errObj, ok := response.(*object.Error); ok {
			return errObj
		}
		body := HTTPClientBodyBuiltin(false).Fn(response)
		key := &object.String{Value: "body"}
		return &object.Dict{Pairs: map[object.DictKey]object.DictPair{key.DictKey(): {Key: key, Value: body}}}
	}}
}

func RegisterRoutesBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 && len(args) != 3 {
			return NewError("registerRoute expects 2 or 3 arguments")
		}
		method := &object.String{Value: "GET"}
		pathIndex, handlerIndex := 0, 1
		if len(args) == 3 {
			method = args[0].(*object.String)
			pathIndex, handlerIndex = 1, 2
		}
		return HTTPRouteBuiltin().Fn(legacyHTTPApp, method, args[pathIndex], args[handlerIndex])
	}}
}

func UseMiddlewaresBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("useMiddleware expects 2 arguments")
		}
		return legacyHTTPApp
	}}
}

func HtmlHandlerBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return NewError("html expects 1 argument")
		}
		content := args[0]
		return &object.Builtin{Fn: func(_ ...object.Object) object.Object {
			return HTTPResponseBuiltin("html").Fn(&object.Integer{Value: 200}, content)
		}}
	}}
}

func ServerStaticBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("serveStatic expects 2 arguments")
		}
		return HTTPStaticBuiltin().Fn(legacyHTTPApp, args[0], args[1])
	}}
}
