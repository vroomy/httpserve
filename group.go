package httpserve

import "path"

func newGroup(r *Router, route string, hs ...Handler) *group {
	var g group
	g.r = r
	g.route = route
	g.hs = append([]Handler{}, hs...)
	return &g
}

// Group represents a handler group
type group struct {
	r     *Router
	route string
	hs    []Handler
}

// GET will set a GET endpoint
func (g *group) GET(route string, hs ...Handler) (err error) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return g.r.GET(route, newHandler(hs))
}

// PUT will set a PUT endpoint
func (g *group) PUT(route string, hs ...Handler) (err error) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return g.r.PUT(route, newHandler(hs))
}

// POST will set a POST endpoint
func (g *group) POST(route string, hs ...Handler) (err error) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return g.r.POST(route, newHandler(hs))
}

// DELETE will set a DELETE endpoint
func (g *group) DELETE(route string, hs ...Handler) (err error) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return g.r.DELETE(route, newHandler(hs))
}

// OPTIONS will set a OPTIONS endpoint
func (g *group) OPTIONS(route string, hs ...Handler) (err error) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return g.r.OPTIONS(route, newHandler(hs))
}

// Handle will create a route for any method
func (g *group) Handle(method, route string, hs ...Handler) (err error) {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return g.r.Handle(method, route, newHandler(hs))
}

// Group will return a new group
func (g *group) Group(route string, hs ...Handler) Group {
	if g.route != "" {
		route = path.Join(g.route, route)
	}

	hs = append(append([]Handler{}, g.hs...), hs...)

	return newGroup(g.r, route, hs...)
}
