package httpserve

import (
	"github.com/julienschmidt/httprouter"
)

// New will return a new instance of Serve
func New() *Serve {
	var s Serve
	s.g.r = httprouter.New()
	return &s
}

// Serve will serve HTTP requests
type Serve struct {
	g Group
}

// GET will set a GET endpoint
func (s *Serve) GET(route string, hs ...Handler) {
	s.g.GET(route, hs...)
}

// PUT will set a PUT endpoint
func (s *Serve) PUT(route string, hs ...Handler) {
	s.g.PUT(route, hs...)
}

// POST will set a POST endpoint
func (s *Serve) POST(route string, hs ...Handler) {
	s.g.POST(route, hs...)
}

// DELETE will set a DELETE endpoint
func (s *Serve) DELETE(route string, hs ...Handler) {
	s.g.DELETE(route, hs...)
}

// Group will return a new group for a given route and handlers
func (s *Serve) Group(route string, hs ...Handler) *Group {
	return s.g.Group(route, hs...)
}
