package httpserve

import (
	"fmt"
	"log"
	"net/http"
)

const (
	colon        = ':'
	forwardSlash = "/"
)

func newRouter() *Router {
	var r Router
	r.rm = make(routesMap, 3)
	r.SetNotFound(notFoundHandler)
	r.SetPanic(r.onPanic)
	return &r
}

// Router handles routes
type Router struct {
	rm routesMap

	notFound Handler
	panic    PanicHandler

	maxParams int
}

func (r *Router) onPanic(v interface{}) {
	log.Println("Panic encountered", v)
}

// Match will check a url for a matching Handler, and return any associated handler and its parameters
func (r *Router) Match(method, url string) (h Handler, p Params, ok bool) {
	var rs routes
	if rs, ok = r.rm[method]; !ok {
		return
	}

	p = make(Params, 0, r.maxParams)
	for _, rt := range rs {
		if p, ok = rt.check(p, url); ok {
			h = rt.h
			return
		}

		p = p[:0]
	}

	// No match was found, set handler as our not found handler
	h = r.notFound
	return
}

// SetNotFound will set the not found handler (404)
func (r *Router) SetNotFound(hs ...Handler) {
	r.notFound = newHandler(hs)
}

// SetPanic will set panic handler
func (r *Router) SetPanic(h PanicHandler) {
	r.panic = h
}

// Handle will create a route for any method
func (r *Router) Handle(method, url string, h Handler) (err error) {
	var route *route
	if route, err = newRoute(url, h, method); err != nil {
		return fmt.Errorf("error creating route for [%s] \"%s\": %v", method, url, err)
	}

	if n := route.numParams(); n > r.maxParams {
		r.maxParams = n
	}

	rs := r.rm[method]
	rs = append(rs, route)
	r.rm[method] = rs
	return
}

// GET will create a GET route
func (r *Router) GET(url string, h Handler) error {
	return r.Handle("GET", url, h)
}

// PUT will create a PUT route
func (r *Router) PUT(url string, h Handler) error {
	return r.Handle("PUT", url, h)
}

// POST will create a POST route
func (r *Router) POST(url string, h Handler) error {
	return r.Handle("POST", url, h)
}

// DELETE will create a DELETE route
func (r *Router) DELETE(url string, h Handler) error {
	return r.Handle("DELETE", url, h)
}

// OPTIONS will create an OPTIONS route
func (r *Router) OPTIONS(url string, h Handler) error {
	return r.Handle("OPTIONS", url, h)
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h, params, ok := r.Match(req.Method, req.URL.Path)
	if !ok {
		h = r.notFound
	}

	ctx := newContext(rw, req, params)

	defer func() {
		if p := recover(); p != nil && r.panic != nil {
			rw.WriteHeader(500)
			r.panic(p)
		}
	}()

	h(ctx)
}
