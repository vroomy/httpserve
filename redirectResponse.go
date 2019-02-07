package httpserve

import (
	"io"
)

// NewRedirectResponse will return a redirect response
func NewRedirectResponse(code int, url string) *RedirectResponse {
	var r RedirectResponse
	r.code = code
	r.url = url
	return &r
}

// RedirectResponse will redirect the client
type RedirectResponse struct {
	code int
	url  string
}

// ContentType returns the content type
func (r *RedirectResponse) ContentType() (contentType string) {
	return ""
}

// StatusCode returns the status code
func (r *RedirectResponse) StatusCode() (code int) {
	return r.code
}

// WriteTo will write to a given io.Writer
func (r *RedirectResponse) WriteTo(w io.Writer) (n int64, err error) {
	return
}
