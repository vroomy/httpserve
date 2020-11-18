package httpserve

import (
	"fmt"
	"net/http"
	"strings"
)

// newHandler will return a new Handler
func newHandler(hs []Handler) Handler {
	return func(ctx *Context) (resp Response) {
		// Get response from context by passing provided handlers
		resp = ctx.getResponse(hs)

		// Check to see if the context was adopted
		if ctx.wasAdopted(resp) {
			return
		}
		defer ctx.Request.Body.Close()

		// Respond using context
		ctx.respond(resp)

		sc := 200
		if resp != nil {
			sc = resp.StatusCode()
		}

		// Process context hooks
		ctx.processHooks(sc)
		return
	}
}

func newHTTPServer(h http.Handler, port uint16, c Config) *http.Server {
	var srv http.Server
	srv.Handler = h
	srv.Addr = fmt.Sprintf(":%d", port)
	srv.ReadTimeout = c.ReadTimeout
	srv.WriteTimeout = c.WriteTimeout
	srv.MaxHeaderBytes = c.MaxHeaderBytes
	return &srv
}

// getParts is used to split URLs into parts
func getParts(url string) (parts []string, err error) {
	if url == "/" {
		parts = []string{"/"}
		return
	}

	var buf []byte
	for _, part := range strings.Split(url, "/") {
		if len(part) == 0 {
			continue
		}

		switch part[0] {
		case ':':
		case '*':

		default:
			buf = append(buf, '/')
			buf = append(buf, part...)
			continue
		}

		if len(buf) > 0 {
			parts = append(parts, string(buf))
			buf = buf[:0]
		}

		parts = append(parts, part)
	}

	if len(buf) == 0 {
		return
	}

	parts = append(parts, string(buf))

	return
}

func isPartMatch(url, part string) (match bool) {
	if len(url) < len(part) {
		// Remaining URL is less than our part, return
		return
	}

	if url == part {
		return true
	}

	if url[:len(part)] != part {
		return
	}

	remaining := url[len(part):]
	return len(remaining) == 0 || remaining[0] == '/'
}

func shiftStr(str string, n int) (out string) {
	switch {
	case len(str) >= n:
		return str[n:]
	case len(str) >= n:
		return str[n:]

	default:
		return str

	}
}

func notFoundHandler(ctx *Context) Response {
	return NewTextResponse(404, []byte("404, not found"))
}

// PanicHandler is a panic handler
type PanicHandler func(v interface{})
