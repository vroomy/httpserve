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
func (r *route) check(url string, p Params) (ok bool) {
	for _, part := range r.s {
		switch {
		case len(url) == 0:
			return

		case part[0] == colon:
			key, value, n := getParamMatch(part, url)
			p[key] = value
			url = shiftStr(url, n+1)

		case part[0] == '*':
			ok = true
			return

		case isPartMatch(url, part):
			// Part matches, increment and move on
			url = shiftStr(url, len(part)+1)

		default:
			// We do not have a match, return
			return
		}
	}

	ok = true
	return
}

func isPartMatch(url, part string) (match bool) {
	if len(url) < len(part) {
		// Remaining URL is less than our part, return
		return
	}

	return url[:len(part)] == part
}

func getParamMatch(part, url string) (key, value string, n int) {
	if n = strings.IndexByte(url, '/'); n == -1 {
		n = len(url)
	}

	key = part[1:]
	value = url[:n]
	return
}

func shiftStr(str string, n int) (out string) {
	if len(str) > n {
		return str[n:]
	}

	return str
}
