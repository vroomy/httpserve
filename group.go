package httpserve

import (
	"path"

	"github.com/hatchify/scribe"
	"github.com/vroomy/common"
)

func newGroup(r *Router, route string, hs ...common.Handler) *group {
	var g group
	g.r = r
	g.route = route
	g.hs = hs
	return &g
}

// Group represents a handler group
type group struct {
	r     *Router
	route string
	hs    []common.Handler
}

// GET will set a GET endpoint
func (g *group) GET(route string, hs ...common.Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.GET(route, newHandler(hs))
}

// PUT will set a PUT endpoint
func (g *group) PUT(route string, hs ...common.Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.PUT(route, newHandler(hs))
}

// POST will set a POST endpoint
func (g *group) POST(route string, hs ...common.Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	scribe.New("httpserve").Notificationf("Setting handlers for route %s:%v", route, hs)

	g.r.POST(route, newHandler(hs))
}

// DELETE will set a DELETE endpoint
func (g *group) DELETE(route string, hs ...common.Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.DELETE(route, newHandler(hs))
}

// OPTIONS will set a OPTIONS endpoint
func (g *group) OPTIONS(route string, hs ...common.Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.OPTIONS(route, newHandler(hs))
}

// Group will return a new group
func (g *group) Group(route string, hs ...common.Handler) Group {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	return newGroup(g.r, route, hs...)
}

// Group is a grouping interface
type Group interface {
	GET(route string, hs ...common.Handler)
	POST(route string, hs ...common.Handler)
	PUT(route string, hs ...common.Handler)
	DELETE(route string, hs ...common.Handler)
	OPTIONS(route string, hs ...common.Handler)

	Group(route string, hs ...common.Handler) Group
}
