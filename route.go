package httpserve

// route shares static prefixes and parameter branches for one HTTP method.
// Static branches take precedence over parameters, followed by wildcards.
type route struct {
	root  routeNode
	count int
}

func (t *route) add(parts []string, handler Handler) {
	t.count++
	var (
		node      = &t.root
		numParams int
	)
	for _, part := range parts {
		switch part[0] {
		case colon:
			numParams++
			if node.param == nil {
				node.param = &routeNode{}
			}

			node = node.param
		case '*':
			// The existing matcher ignores everything following a wildcard.
			if node.wildcard == nil {
				node.wildcard = newRouteEndpoint(handler, parts, numParams)
			}

			return
		default:
			node = node.insertStatic(part)
		}
	}

	if node.endpoint == nil {
		node.endpoint = newRouteEndpoint(handler, parts, numParams)
	}
}

// find appends captured parameters to borrowed storage only after a complete
// match. A miss leaves the storage unchanged, including its backing array.
func (t *route) find(url string, params *Params) *routeEndpoint {
	if t.count == 0 {
		return nil
	}

	var (
		start    = len(*params)
		endpoint = t.root.find(url, params, start)
	)
	if endpoint == nil {
		return nil
	}

	for index, name := range endpoint.paramNames {
		(*params)[start+index].Key = name
	}

	return endpoint
}
