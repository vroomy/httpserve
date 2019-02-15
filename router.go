package httpserve

import (
	"net/http"
)

const (
	colon        = ':'
	forwardSlash = "/"
)

func newRouter() *Router {
	var r Router
	r.rm = make(routesMap, 3)
	r.notFound = notFoundHandler
	return &r
}

// Router handles routes
type Router struct {
	rm routesMap

	notFound  Handler
	maxParams int
}

// Match will check a url for a matching Handler, and return any associated handler and it's parameters
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
func (r *Router) SetNotFound(h Handler) {
	r.notFound = h
}

// Handle will create a route for any method
func (r *Router) Handle(method, url string, h Handler) {
	route := newRoute(url, h, method)

	if n := route.numParams(); n > r.maxParams {
		r.maxParams = n
	}

	rs := r.rm[method]
	rs = append(rs, route)
	r.rm[method] = rs
}

// GET will create a GET route
func (r *Router) GET(url string, h Handler) {
	r.Handle("GET", url, h)
}

// PUT will create a PUT route
func (r *Router) PUT(url string, h Handler) {
	r.Handle("PUT", url, h)
}

// POST will create a POST route
func (r *Router) POST(url string, h Handler) {
	r.Handle("POST", url, h)
}

// DELETE will create a DELETE route
func (r *Router) DELETE(url string, h Handler) {
	r.Handle("DELETE", url, h)
}

// OPTIONS will create an OPTIONS route
func (r *Router) OPTIONS(url string, h Handler) {
	r.Handle("OPTIONS", url, h)
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h, params, ok := r.Match(req.Method, req.URL.Path)
	if !ok {
		h = r.notFound
	}

	ctx := newContext(rw, req, params)
	resp := h(ctx)
	ctx.respond(resp)
}
