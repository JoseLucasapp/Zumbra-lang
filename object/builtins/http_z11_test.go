package builtins

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"zumbra/object"
)

func z11TestDict(values map[string]object.Object) *object.Dict {
	pairs := make(map[object.DictKey]object.DictPair, len(values))
	for key, value := range values {
		objectKey := NewString(key)
		pairs[objectKey.DictKey()] = object.DictPair{Key: objectKey, Value: value}
	}
	return &object.Dict{Pairs: pairs}
}

func requireHTTPValue(t *testing.T, value object.Object) object.Object {
	t.Helper()
	if err, ok := value.(*object.Error); ok {
		t.Fatalf("HTTP builtin failed: %s", err.Message)
	}
	return value
}

func TestZ11HTTPRouterMiddlewareJSONCookiesAndClient(t *testing.T) {
	app := requireHTTPValue(t, HTTPAppBuiltin().Fn()).(*object.HttpApp)
	middleware := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		response := args[1].(*object.HttpResponse)
		response.Headers["X-Zumbra"] = []string{"0.7.0"}
		return NewBoolean(true)
	}}
	handler := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		request := args[0].(*object.HttpRequest)
		name, ok := httpDictString(request.Params, "name")
		if !ok {
			return NewError("route parameter is missing")
		}
		body := z11TestDict(map[string]object.Object{
			"message": NewString("hello " + name),
			"method":  NewString(request.Method),
		})
		response := requireHTTPValue(t, HTTPResponseBuiltin("json").Fn(NewInteger(200), body)).(*object.HttpResponse)
		return requireHTTPValue(t, HTTPCookieBuiltin().Fn(response, NewString("session"), NewString("local"), z11TestDict(map[string]object.Object{"httpOnly": NewBoolean(true)})))
	}}
	requireHTTPValue(t, HTTPUseBuiltin().Fn(app, middleware))
	requireHTTPValue(t, HTTPRouteBuiltin().Fn(app, NewString("GET"), NewString("/hello/:name"), handler))
	server := requireHTTPValue(t, HTTPServeBuiltin().Fn(app, NewString("127.0.0.1"), NewInteger(0))).(*object.HttpServer)
	defer HTTPShutdownBuiltin().Fn(server, NewInteger(2000))

	port := requireHTTPValue(t, HTTPServerPortBuiltin().Fn(server)).(*object.Integer).Value
	target := NewString("http://127.0.0.1:" + toDecimal(port) + "/hello/zumbra")
	response := requireHTTPValue(t, HTTPRequestBuiltin().Fn(NewString("GET"), target, z11TestDict(nil), NewString(""), NewInteger(2000))).(*object.HttpClientResponse)
	if response.StatusCode != 200 {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	if header, ok := httpDictString(response.Headers, "x-zumbra"); !ok || header != "0.7.0" {
		t.Fatalf("middleware header was not preserved: %s", response.Headers.Inspect())
	}
	if cookie, ok := httpDictString(response.Cookies, "session"); !ok || cookie != "local" {
		t.Fatalf("cookie was not returned: %s", response.Cookies.Inspect())
	}
	decoded := requireHTTPValue(t, HTTPClientJSONBuiltin().Fn(response)).(*object.Dict)
	if message, ok := httpDictString(decoded, "message"); !ok || message != "hello zumbra" {
		t.Fatalf("unexpected JSON body: %s", decoded.Inspect())
	}
}

func TestZ11JSONJWTAndSSE(t *testing.T) {
	payload := z11TestDict(map[string]object.Object{"name": NewString("zumbra"), "value": NewInteger(42)})
	encoded := requireHTTPValue(t, JSONStringifyBuiltin().Fn(payload)).(*object.String)
	decoded := requireHTTPValue(t, JSONParseZ11Builtin().Fn(encoded)).(*object.Dict)
	if name, ok := httpDictString(decoded, "name"); !ok || name != "zumbra" {
		t.Fatalf("JSON round-trip failed: %s", decoded.Inspect())
	}

	token := requireHTTPValue(t, JWTSignHS256Builtin().Fn(z11TestDict(map[string]object.Object{"sub": NewString("local")}), NewString("secret"), NewInteger(60))).(*object.String)
	verified := requireHTTPValue(t, JWTVerifyHS256Builtin().Fn(token, NewString("secret"))).(*object.Array)
	if !verified.Elements[0].(*object.Boolean).Value {
		t.Fatalf("JWT verification failed: %s", verified.Inspect())
	}
	rejected := requireHTTPValue(t, JWTVerifyHS256Builtin().Fn(token, NewString("wrong"))).(*object.Array)
	if rejected.Elements[0].(*object.Boolean).Value {
		t.Fatal("JWT with wrong secret was accepted")
	}

	event := requireHTTPValue(t, SSEEventBuiltin().Fn(NewString("ready"), NewString("status"), NewString("1"), NewInteger(1000))).(*object.String)
	for _, expected := range []string{"event: status", "id: 1", "retry: 1000", "data: ready"} {
		if !strings.Contains(event.Value, expected) {
			t.Fatalf("SSE event missing %q: %q", expected, event.Value)
		}
	}
}

func TestZ11GracefulShutdownStopsServer(t *testing.T) {
	app := object.NewHttpApp()
	requireHTTPValue(t, HTTPRouteBuiltin().Fn(app, NewString("GET"), NewString("/"), &object.Builtin{Fn: func(args ...object.Object) object.Object {
		return HTTPResponseBuiltin("text").Fn(NewInteger(200), NewString("ok"))
	}}))
	server := requireHTTPValue(t, HTTPServeBuiltin().Fn(app, NewString("127.0.0.1"), NewInteger(0))).(*object.HttpServer)
	if !server.Running.Load() {
		t.Fatal("server was not marked as running")
	}
	result := requireHTTPValue(t, HTTPShutdownBuiltin().Fn(server, NewInteger(2000))).(*object.Boolean)
	if !result.Value {
		t.Fatal("graceful shutdown failed")
	}
	deadline := time.Now().Add(time.Second)
	for server.Running.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if server.Running.Load() {
		t.Fatal("server remained running after shutdown")
	}
}

func toDecimal(value int64) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}

func TestZ11MultipartUploadAndBodyLimit(t *testing.T) {
	app := object.NewHttpApp()
	handler := &object.Builtin{Fn: func(args ...object.Object) object.Object {
		request := args[0].(*object.HttpRequest)
		pair, ok := request.Files.Pairs[NewString("upload").DictKey()]
		if !ok {
			return NewError("multipart file is missing")
		}
		file, ok := pair.Value.(*object.HttpUploadedFile)
		if !ok {
			return NewError("unexpected uploaded file object %T", pair.Value)
		}
		label, _ := httpDictString(request.Form, "label")
		return HTTPResponseBuiltin("json").Fn(NewInteger(200), z11TestDict(map[string]object.Object{
			"filename": NewString(file.Filename),
			"size":     NewInteger(int64(len(file.Data))),
			"label":    NewString(label),
		}))
	}}
	requireHTTPValue(t, HTTPRouteBuiltin().Fn(app, NewString("POST"), NewString("/upload"), handler))
	server := requireHTTPValue(t, HTTPServeBuiltin().Fn(app, NewString("127.0.0.1"), NewInteger(0))).(*object.HttpServer)
	defer HTTPShutdownBuiltin().Fn(server, NewInteger(2000))
	port := requireHTTPValue(t, HTTPServerPortBuiltin().Fn(server)).(*object.Integer).Value

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("label", "cover"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("upload", "game.nes")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte{0x4e, 0x45, 0x53, 0x1a}); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:"+toDecimal(port)+"/upload", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("unexpected upload status %d", response.StatusCode)
	}

	requireHTTPValue(t, HTTPLimitBodyBuiltin().Fn(app, NewInteger(2)))
	oversized, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:"+toDecimal(port)+"/upload", strings.NewReader("larger"))
	if err != nil {
		t.Fatal(err)
	}
	response, err = http.DefaultClient.Do(oversized)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", response.StatusCode)
	}
}
