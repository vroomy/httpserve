package httpserve

import (
	"io"
)

// NewXMLResponse will return a new text response
func NewXMLResponse(code int, value []byte) *XMLResponse {
	var j XMLResponse
	j.code = code
	j.val = value
	return &j
}

// XMLResponse is a basic text response
type XMLResponse struct {
	code int
	val  []byte
}

// ContentType returns the content type
func (j *XMLResponse) ContentType() (contentType string) {
	return "text/xml"
}

// StatusCode returns the status code
func (j *XMLResponse) StatusCode() (code int) {
	return j.code
}

// WriteTo will write to a given io.Writer
func (j *XMLResponse) WriteTo(w io.Writer) (n int64, err error) {
	var nint int
	nint, err = w.Write(j.val)
	n = int64(nint)
	return
}
