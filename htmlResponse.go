package httpserve

import (
	"bytes"
	"io"
)

// NewHTMLResponse will return a new text response
func NewHTMLResponse(code int, body []byte) *HTMLResponse {
	var h HTMLResponse
	h.code = code
	h.r = bytes.NewReader(body)
	return &h
}

// HTMLResponse is a basic text response
type HTMLResponse struct {
	code int
	r    io.Reader
}

// ContentType returns the content type
func (h *HTMLResponse) ContentType() (contentType string) {
	return "text/html"
}

// StatusCode returns the status code
func (h *HTMLResponse) StatusCode() (code int) {
	return h.code
}

// WriteTo will write the internal reader to a provided io.Writer
func (h *HTMLResponse) WriteTo(w io.Writer) (n int64, err error) {
	return io.Copy(w, h.r)
}
