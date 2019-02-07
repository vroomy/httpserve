package httpserve

import "path"

func newGroup(r *Router, route string, hs ...Handler) *Group {
	var g Group
	g.r = r
	g.route = route
	g.hs = hs
	return &g
}

// Group represents a handler group
type Group struct {
	r     *Router
	route string
	hs    []Handler
}

// GET will set a GET endpoint
func (g *Group) GET(route string, hs ...Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.GET(route, newHandler(hs))
}

// PUT will set a PUT endpoint
func (g *Group) PUT(route string, hs ...Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.PUT(route, newHandler(hs))
}

// POST will set a POST endpoint
func (g *Group) POST(route string, hs ...Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.POST(route, newHandler(hs))
}

// DELETE will set a DELETE endpoint
func (g *Group) DELETE(route string, hs ...Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.DELETE(route, newHandler(hs))
}

// OPTIONS will set a OPTIONS endpoint
func (g *Group) OPTIONS(route string, hs ...Handler) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	g.r.OPTIONS(route, newHandler(hs))
}

// Group will return a new group
func (g *Group) Group(route string, hs ...Handler) *Group {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	if len(g.hs) > 0 {
		hs = append(g.hs, hs...)
	}

	return newGroup(g.r, route, hs...)
}
