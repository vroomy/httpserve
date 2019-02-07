package httpserve

import (
	"net/http"
)

const (
	colon        = ':'
	forwardSlash = "/"
)

type method uint8

const (
	methodNil method = iota
	methodGET
	methodPUT
	methodPOST
	methodDELETE
	methodOPTIONS
)

func newRouter() *Router {
	var r Router
	r.routes = make([]*route, 0)
	r.notFound = notFoundHandler
	return &r
}

// Router handles routes
type Router struct {
	routes   []*route
	notFound Handler

	maxParams int
}

// Match will check a url for a matching Handler, and return any associated handler and it's parameters
func (r *Router) Match(url string) (h Handler, p Params, ok bool) {
	p = make(Params, 0, r.maxParams)
	for _, rt := range r.routes {
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

func (r *Router) set(route *route) {
	if n := route.numParams(); n > r.maxParams {
		r.maxParams = n
	}

	r.routes = append(r.routes, route)
}

// GET will create a GET route
func (r *Router) GET(url string, h Handler) {
	r.set(newRoute(url, h, methodGET))
}

// PUT will create a PUT route
func (r *Router) PUT(url string, h Handler) {
	r.set(newRoute(url, h, methodPUT))
}

// POST will create a POST route
func (r *Router) POST(url string, h Handler) {
	r.set(newRoute(url, h, methodPOST))
}

// DELETE will create a DELETE route
func (r *Router) DELETE(url string, h Handler) {
	r.set(newRoute(url, h, methodDELETE))
}

// OPTIONS will create an OPTIONS route
func (r *Router) OPTIONS(url string, h Handler) {
	r.set(newRoute(url, h, methodOPTIONS))
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h, params, ok := r.Match(req.URL.Path)
	if !ok {
		h = r.notFound
	}

	ctx := newContext(rw, req, params)
	resp := h(ctx)
	ctx.respond(resp)
}
