package httpserve

import (
	"fmt"
	"net/http"
	"strings"
)

func newHTTPServer(h http.Handler, port uint16, c Config) *http.Server {
	var srv http.Server
	srv.Handler = h
	srv.Addr = fmt.Sprintf(":%d", port)
	srv.ReadTimeout = c.ReadTimeout
	srv.WriteTimeout = c.WriteTimeout
	srv.IdleTimeout = c.IdleTimeout
	srv.MaxHeaderBytes = c.MaxHeaderBytes
	srv.DisableGeneralOptionsHandler = true
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

func notFoundHandler(ctx *Context) {
	ctx.WriteString(404, "text/plain", "404, not found")
}
