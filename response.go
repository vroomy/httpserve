package httpserve

import "io"

// Response is the http response handlers return
type Response interface {
	StatusCode() (code int)
	ContentType() (contentType string)
	WriteTo(w io.Writer) (n int64, err error)
}
