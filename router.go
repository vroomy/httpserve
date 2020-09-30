package httpserve

import (
	"fmt"
	"log"
	"net/http"

	"github.com/vroomy/common"
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
	getRoutes    routes
	putRoutes    routes
	postRoutes   routes
	deleteRoutes routes
	optionRoutes routes

	rm routesMap

	notFound common.Handler
	panic    PanicHandler

	maxParams int
}

func (r *Router) onPanic(v interface{}) {
	log.Println("Panic encountered", v)
}

// Match will check a url for a matching Handler, and return any associated handler and its parameters
func (r *Router) Match(method, url string) (h common.Handler, p Params, ok bool) {
	var rs routes

	switch method {
	case http.MethodGet:
		rs = r.getRoutes
	case http.MethodPut:
		rs = r.putRoutes
	case http.MethodPost:
		rs = r.postRoutes
	case http.MethodDelete:
		rs = r.deleteRoutes
	case http.MethodOptions:
		rs = r.optionRoutes

	default:
		if rs, ok = r.rm[method]; !ok {
			return
		}
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
func (r *Router) SetNotFound(hs ...common.Handler) {
	r.notFound = newHandler(hs)
}

// SetPanic will set panic handler
func (r *Router) SetPanic(h PanicHandler) {
	r.panic = h
}

// Handle will create a route for any method
func (r *Router) Handle(method, url string, h common.Handler) (err error) {
	var route *route
	if route, err = newRoute(url, h, method); err != nil {
		return fmt.Errorf("error creating route for [%s] \"%s\": %v", method, url, err)
	}

	if n := route.numParams(); n > r.maxParams {
		r.maxParams = n
	}

	switch method {
	case http.MethodGet:
		r.getRoutes = append(r.getRoutes, route)
	case http.MethodPut:
		r.putRoutes = append(r.putRoutes, route)
	case http.MethodPost:
		r.postRoutes = append(r.postRoutes, route)
	case http.MethodDelete:
		r.deleteRoutes = append(r.deleteRoutes, route)
	case http.MethodOptions:
		r.optionRoutes = append(r.optionRoutes, route)

	default:
		rs := r.rm[method]
		rs = append(rs, route)
		r.rm[method] = rs
	}

	return
}

// GET will create a GET route
func (r *Router) GET(url string, h common.Handler) error {
	return r.Handle("GET", url, h)
}

// PUT will create a PUT route
func (r *Router) PUT(url string, h common.Handler) error {
	return r.Handle("PUT", url, h)
}

// POST will create a POST route
func (r *Router) POST(url string, h common.Handler) error {
	return r.Handle("POST", url, h)
}

// DELETE will create a DELETE route
func (r *Router) DELETE(url string, h common.Handler) error {
	return r.Handle("DELETE", url, h)
}

// OPTIONS will create an OPTIONS route
func (r *Router) OPTIONS(url string, h common.Handler) error {
	return r.Handle("OPTIONS", url, h)
}

// ROUTE will create a route with any given method
func (r *Router) ROUTE(method, url string, h common.Handler) error {
	return r.Handle(method, url, h)
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
