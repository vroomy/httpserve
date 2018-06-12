package httpserve

import (
	"strings"
)

func newRoute(url string, h Handler, m method) *route {
	if url[0] != '/' {
		panic("invalid route, needs to start with a forward slash")
	}

	var r route
	r.s = getParts(url)
	r.m = m
	r.h = h
	return &r
}

type route struct {
	s []string
	h Handler

	m method
}

// check will check a url for a match, it will also return any associated parameters
func (r *route) check(url string) (p Params, ok bool) {
	var lastIndex, urlIndex int
	p = make(Params, 1)

	for i, part := range r.s {
		if part[0] == colon {
			key := part[1:]
			urlIndex++
			nextSlash := strings.IndexByte(url[urlIndex:], '/')
			if nextSlash == -1 {
				nextSlash = len(url)
			}

			p[key] = url[urlIndex:nextSlash]
			lastIndex += len(part)
			urlIndex += nextSlash
			continue
		}

		if part[0] == '*' {
			ok = true
			return
		}

		if len(url[urlIndex:]) < len(part) {
			return
		}

		if url[urlIndex:i+len(part)] != part {
			// We do not have a match, bail out
			return
		}

		lastIndex += len(part)
		urlIndex += len(part)
	}

	ok = lastIndex == len(url)
	return
}
