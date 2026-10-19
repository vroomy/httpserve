package apex

import "net/http"

// routerBenchmarkWriter discards output and can be reused without growing a body
// buffer. Request creation, network I/O, and response recording are not timed.
type routerBenchmarkWriter struct {
	header http.Header
	status int
}

func (w *routerBenchmarkWriter) Header() http.Header {
	return w.header
}

func (w *routerBenchmarkWriter) WriteHeader(status int) {
	w.status = status
}

func (w *routerBenchmarkWriter) Write(body []byte) (int, error) {
	return len(body), nil
}
