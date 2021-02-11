package httpserve

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/vroomy/common"
)

// newHandler will return a new common.Handler
func newHandler(hs []common.Handler) common.Handler {
	return func(ctx common.Context) {
		c, ok := ctx.(*Context)
		if !ok {
			panic(fmt.Sprintf("invalid context type, %T is not supported", ctx))
		}

		// Process handler funcs
		c.processHandlers(hs)
		if c.completed {
			c.request.Body.Close()
		}

		// Process context hooks
		c.processHooks()
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

func notFoundHandler(ctx common.Context) {
	ctx.WriteString(404, "text/plain", "404, not found")
}

// PanicHandler is a panic handler
type PanicHandler func(v interface{})

type redirectQuery struct {
	Redirect string `form:"redirect"`
}

func getFuncName(fn interface{}) string {
	val := reflect.ValueOf(fn).Pointer()
	return runtime.FuncForPC(val).Name()
}
