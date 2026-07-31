package object

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

// HttpRoute is an immutable route registration owned by an HttpApp.
type HttpRoute struct {
	Method  string
	Pattern string
	Handler Object
}

type HttpStaticRoute struct {
	Prefix    string
	Directory string
}

type HttpCorsConfig struct {
	Enabled          bool
	Origins          []string
	Methods          []string
	Headers          []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

// HttpApp owns all server configuration. Unlike the legacy HTTP builtins, state
// is not global, so multiple independent applications can run in one process.
type HttpApp struct {
	Mu           sync.RWMutex
	Routes       []HttpRoute
	Middlewares  []Object
	StaticRoutes []HttpStaticRoute
	MaxBodyBytes int64
	Compression  bool
	CORS         HttpCorsConfig
}

func NewHttpApp() *HttpApp {
	return &HttpApp{MaxBodyBytes: 8 * 1024 * 1024}
}
func (a *HttpApp) Type() ObjectType { return HTTP_APP_OBJ }
func (a *HttpApp) Inspect() string {
	if a == nil {
		return "HttpApp<nil>"
	}
	a.Mu.RLock()
	defer a.Mu.RUnlock()
	return fmt.Sprintf("HttpApp<routes=%d middleware=%d>", len(a.Routes), len(a.Middlewares))
}

// HttpServer is a non-blocking HTTP server handle returned by httpServe.
type HttpServer struct {
	Server   *http.Server
	Listener net.Listener
	Done     chan struct{}
	Err      atomic.Value // stores error
	Running  atomic.Bool
	Once     sync.Once
}

func (s *HttpServer) Type() ObjectType { return HTTP_SERVER_OBJ }
func (s *HttpServer) Inspect() string {
	if s == nil || s.Listener == nil {
		return "HttpServer<nil>"
	}
	return fmt.Sprintf("HttpServer<%s running=%t>", s.Listener.Addr(), s.Running.Load())
}

// HttpUploadedFile represents one multipart file entirely held in memory. The
// server enforces the application's body limit before constructing it.
type HttpUploadedFile struct {
	FieldName   string
	Filename    string
	ContentType string
	Data        []byte
}

func (f *HttpUploadedFile) Type() ObjectType { return HTTP_FILE_OBJ }
func (f *HttpUploadedFile) Inspect() string {
	if f == nil {
		return "HttpFile<nil>"
	}
	return fmt.Sprintf("HttpFile<%s %d bytes>", f.Filename, len(f.Data))
}

// HttpRequest is the server-side request representation exposed to Zumbra.
type HttpRequest struct {
	Method        string
	Scheme        string
	Host          string
	Path          string
	RemoteAddress string
	Params        *Dict
	Query         *Dict
	Headers       *Dict
	Cookies       *Dict
	Form          *Dict
	Files         *Dict
	Body          Object
	RawBody       string
	RawBytes      []byte

	Writer   http.ResponseWriter
	Request  *http.Request
	Hijacked atomic.Bool
}

func (r *HttpRequest) Type() ObjectType { return HTTP_REQUEST_OBJ }
func (r *HttpRequest) Inspect() string {
	if r == nil {
		return "HttpRequest<nil>"
	}
	return fmt.Sprintf("HttpRequest<%s %s>", r.Method, r.Path)
}

// HttpResponse is a server-side response value. Headers are multi-value so
// Set-Cookie and other repeated fields remain standards compliant.
type HttpResponse struct {
	StatusCode  int
	Headers     map[string][]string
	Body        Object
	RawBody     []byte
	ContentType string
	Stream      *HttpStream
	Written     bool
}

func NewHttpResponse() *HttpResponse {
	return &HttpResponse{StatusCode: 200, Headers: map[string][]string{}}
}
func (r *HttpResponse) Type() ObjectType { return HTTP_RESPONSE_OBJ }
func (r *HttpResponse) Inspect() string {
	if r == nil {
		return "HttpResponse<nil>"
	}
	return fmt.Sprintf("HttpResponse<status=%d>", r.StatusCode)
}

// HttpClientResponse is immutable data returned by the HTTP client.
type HttpClientResponse struct {
	StatusCode int
	Status     string
	URL        string
	Headers    *Dict
	Cookies    *Dict
	Body       []byte
}

func (r *HttpClientResponse) Type() ObjectType { return HTTP_CLIENT_RESPONSE_OBJ }
func (r *HttpClientResponse) Inspect() string {
	if r == nil {
		return "HttpClientResponse<nil>"
	}
	return fmt.Sprintf("HttpClientResponse<%d %s>", r.StatusCode, r.URL)
}

// HttpStream is a chunked response backed by a Zumbra Channel. A stream may be
// a generic byte stream or Server-Sent Events.
type HttpStream struct {
	Channel     *Channel
	ContentType string
	SSE         bool
}

func (s *HttpStream) Type() ObjectType { return HTTP_STREAM_OBJ }
func (s *HttpStream) Inspect() string {
	if s == nil || s.Channel == nil {
		return "HttpStream<nil>"
	}
	kind := "chunked"
	if s.SSE {
		kind = "sse"
	}
	return fmt.Sprintf("HttpStream<%s>", kind)
}

// WebSocket is an RFC 6455 connection. Conn is protected so concurrent writers
// cannot interleave frames. Reads should be performed by one consumer.
type WebSocket struct {
	Conn      net.Conn
	Reader    *bufio.Reader
	Client    bool
	ReadMu    sync.Mutex
	WriteMu   sync.Mutex
	Closed    atomic.Bool
	CloseOnce sync.Once
}

func (w *WebSocket) Type() ObjectType { return WEB_SOCKET_OBJ }
func (w *WebSocket) Inspect() string {
	if w == nil || w.Conn == nil {
		return "WebSocket<nil>"
	}
	return fmt.Sprintf("WebSocket<%s closed=%t>", w.Conn.RemoteAddr(), w.Closed.Load())
}
