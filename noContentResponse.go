package httpserve

import (
	"io"
)

// NewNoContentResponse will return a new text response
func NewNoContentResponse() *NoContentResponse {
	var t NoContentResponse
	return &t
}

// NoContentResponse is a basic text response
type NoContentResponse struct {
}

// ContentType returns the content type
func (t *NoContentResponse) ContentType() (contentType string) {
	return "text/plain"
}

// StatusCode returns the status code
func (t *NoContentResponse) StatusCode() (code int) {
	return 204
}

// WriteTo will write the internal reader to a provided io.Writer
func (t *NoContentResponse) WriteTo(w io.Writer) (n int64, err error) {
	return
}
