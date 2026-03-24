package object

import "fmt"

type HttpRequest struct {
	Method  string
	Path    string
	Params  *Dict
	Query   *Dict
	Headers *Dict
	Body    Object
	RawBody string
}

func (r *HttpRequest) Type() ObjectType { return "HTTP_REQUEST" }
func (r *HttpRequest) Inspect() string {
	return fmt.Sprintf("HttpRequest<%s %s>", r.Method, r.Path)
}

type HttpResponse struct {
	StatusCode  int
	Headers     map[string]string
	Body        Object
	ContentType string
	Written     bool
}

func NewHttpResponse() *HttpResponse {
	return &HttpResponse{StatusCode: 200, Headers: map[string]string{}}
}

func (r *HttpResponse) Type() ObjectType { return "HTTP_RESPONSE" }
func (r *HttpResponse) Inspect() string {
	return fmt.Sprintf("HttpResponse<status=%d>", r.StatusCode)
}
