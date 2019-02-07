package httpserve

import (
	"io"
)

// NewAdoptResponse will return a new adopt response
func NewAdoptResponse() *AdoptResponse {
	var t AdoptResponse
	return &t
}

// AdoptResponse is a basic adopt response
type AdoptResponse struct {
}

// ContentType returns the content type
func (t *AdoptResponse) ContentType() (contentType string) {
	return "text/plain"
}

// StatusCode returns the status code
func (t *AdoptResponse) StatusCode() (code int) {
	return 200
}

// WriteTo will write the internal reader to a provided io.Writer
func (t *AdoptResponse) WriteTo(w io.Writer) (n int64, err error) {
	return
}
