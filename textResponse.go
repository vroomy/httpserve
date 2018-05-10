package httpserve

import (
	"bytes"
	"io"
)

// NewTextResponse will return a new text response
func NewTextResponse(code int, body []byte) *TextResponse {
	var t TextResponse
	t.code = code
	t.r = bytes.NewReader(body)
	return &t
}

// TextResponse is a basic text response
type TextResponse struct {
	code int
	r    io.Reader
}

// ContentType returns the content type
func (t *TextResponse) ContentType() (contentType string) {
	return "text/plain"
}

// Status returns the status
func (t *TextResponse) Status() (code int) {
	return t.code
}

// WriteTo will write the internal reader to a provided io.Writer
func (t *TextResponse) WriteTo(w io.Writer) (n int64, err error) {
	return io.Copy(w, t.r)
}
