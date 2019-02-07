package httpserve

import (
	"fmt"
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
}

// Match will check a url for a matching Handler, and return any associated handler and it's parameters
func (r *Router) Match(url string) (h Handler, p Params, ok bool) {
	fmt.Println("Matching", url)
	for _, rt := range r.routes {
		fmt.Println("Checking route..", rt.s)
		if p, ok = rt.check(url); ok {
			h = rt.h
			return
		}
	}

	// No match was found, set handler as our not found handler
	h = r.notFound
	return
}

// SetNotFound will set the not found handler (404)
func (r *Router) SetNotFound(h Handler) {
	r.notFound = h
}

// GET will create a GET route
func (r *Router) GET(url string, h Handler) {
	r.routes = append(r.routes, newRoute(url, h, methodGET))
	fmt.Println("GET:", url, r.routes)
}

// PUT will create a PUT route
func (r *Router) PUT(url string, h Handler) {
	r.routes = append(r.routes, newRoute(url, h, methodPUT))
}

// POST will create a POST route
func (r *Router) POST(url string, h Handler) {
	r.routes = append(r.routes, newRoute(url, h, methodPOST))
}

// DELETE will create a DELETE route
func (r *Router) DELETE(url string, h Handler) {
	r.routes = append(r.routes, newRoute(url, h, methodDELETE))
}

// OPTIONS will create an OPTIONS route
func (r *Router) OPTIONS(url string, h Handler) {
	r.routes = append(r.routes, newRoute(url, h, methodOPTIONS))
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h, params, ok := r.Match(req.URL.Path)
	fmt.Println("Hmm", req.URL.Path, ok)

	ctx := newContext(rw, req, params)
	resp := h(ctx)
	ctx.respond(resp)
}
