package httpserve

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

func (r *route) numParams() (n int) {
	for _, part := range r.s {
		if part[0] != colon {
			continue
		}

		n++
	}

	return
}

// check will check a url for a match, it will also return any associated parameters
func (r *route) check(p Params, url string) (out Params, ok bool) {
	out = p

	for _, part := range r.s {
		switch {
		case len(url) == 0:
			return

		case part[0] == colon:
			param, n := newParam(part, url)
			out = append(out, param)
			url = shiftStr(url, n)

		case part[0] == '*':
			ok = true
			return

		case isPartMatch(url, part):
			// Part matches, increment and move on
			url = shiftStr(url, len(part))

		default:
			// We do not have a match, return
			return
		}
	}

	ok = len(url) == 0
	return
}

func isPartMatch(url, part string) (match bool) {
	if len(url) < len(part) {
		// Remaining URL is less than our part, return
		return
	}

	return url[:len(part)] == part
}

func shiftStr(str string, n int) (out string) {
	switch {
	case len(str) >= n+1:
		return str[n+1:]
	case len(str) >= n:
		return str[n:]

	default:
		return str

	}
}
