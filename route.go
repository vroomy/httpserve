package httpserve

func newRoute(url string, h Handler, method string) (rp *route, err error) {
	if url[0] != '/' {
		err = ErrMissingLeadSlash
		return
	}

	var r route
	if r.s, err = getParts(url); err != nil {
		return
	}

	r.method = method
	r.h = h
	rp = &r
	return
}

type route struct {
	s []string
	h Handler

	method string
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
			// Skip forward to avoid slash
			param, n := newParam(part, url[1:])
			// Increment N to account for skipping
			n++
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

type routes []*route

type routesMap map[string]routes
