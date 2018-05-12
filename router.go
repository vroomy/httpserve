package httpserve

import (
	"strings"
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
)

func newRouter() *Router {
	var r Router
	r.routes = make([]*route, 0)
	return &r
}

// Router handle routes
type Router struct {
	routes []*route
}

func (r *Router) check(url string) (match *route, p Params) {
	var ok bool
	for _, rt := range r.routes {
		if p, ok = rt.check(url); ok {
			match = rt
			return
		}
	}

	return
}

// GET will create a GET route
func (r *Router) GET(url string, h Handler) {
	r.routes = append(r.routes, newRoute(url, h, methodGET))
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

func newRoute(url string, h Handler, m method) *route {
	if url[0] != '/' {
		panic("invalid route, needs to start with a forward slash")
	}

	var r route
	r.s = strings.Split(url, forwardSlash)[1:]
	r.m = m
	r.h = h
	return &r
}

type route struct {
	s []string
	h Handler

	m method
}

func (r *route) check(url string) (p Params, ok bool) {
	var lastIndex int
	p = make(Params, 1)

	for i, part := range r.s {
		if part[0] == colon {
			key := part[1:]
			p[key] = part
			lastIndex += len(part)
			continue
		}

		if part[0] == '*' {
			ok = true
			return
		}

		if url[lastIndex:i] != part {
			// We do not have  amatch, bail out
			return
		}

		return
	}

	ok = true
	return
}

// Params represent route params
type Params map[string]string
