package httpserve

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/vroomy/common"
)

// newHandler will return a new Handler
func newHandler(hs []common.Handler) common.Handler {
	return func(context common.Context) (resp common.Response) {
		var ok bool
		var ctx *Context
		if ctx, ok = context.(*Context); !ok {
			return
		}

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

	if url[:len(part)] != part {
		return
	}

	nextSlash := strings.Index(url[len(part):], "/")
	return nextSlash <= 0
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

func notFoundHandler(ctx common.Context) common.Response {
	return NewTextResponse(404, []byte("404, not found"))
}

// PanicHandler is a panic handler
type PanicHandler func(v interface{})
