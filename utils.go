package httpserve

import (
	"fmt"
	"io"
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

// Handler is the HTTP handler type
type Handler func(ctx *Context) Response

// Response is a response interface
type Response interface {
	StatusCode() (code int)
	ContentType() (contentType string)
	WriteTo(w io.Writer) (n int64, err error)
}

// Storage is used as a basic form of Param storage for a Context
// TODO: Determine with team if it seems valuable to change this to map[string]interface{}.
// I'd prefer if we can keep it as-is, due to the fact that map[string]string has much less
// GC overhead. Additionally, avoiding type assertion would be fantastic.
type Storage map[string]string

// Hook is a function called after the response has been completed to the requester
type Hook func(statusCode int, storage Storage)

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
	var (
		lastIndex int
		lastSlash int
	)

	parts = make([]string, 0, 3)

	for i := 0; i < len(url); i++ {
		b := url[i]
		switch b {
		case ':':
			if lastSlash != i-1 {
				err = ErrInvalidParamLocation
				return
			}

			if part := url[lastIndex : i-1]; len(part) > 0 {
				parts = append(parts, part)
			}

			lastIndex = i
		case '*':
			if lastSlash != i-1 {
				// Note: We may want to disable this check if we want to support things like "*.jpg"
				// TODO: Research if we want to support more extendable wildcards
				err = ErrInvalidWildcardLocation
				return
			}

			if part := url[lastIndex : i-1]; len(part) > 0 {
				parts = append(parts, part)
			}

			lastIndex = i

			if len(url)-1 > i {
				err = ErrInvalidWildcardRoute
				return
			}
		case '/':
			lastSlash = i
		}
	}

	if len(url) <= lastIndex {
		return
	}

	parts = append(parts, url[lastIndex:])
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
	case len(str) >= n+1:
		return str[n+1:]
	case len(str) >= n:
		return str[n:]

	default:
		return str

	}
}

func notFoundHandler(ctx *Context) Response {
	return NewTextResponse(404, []byte("404, not found"))
}
