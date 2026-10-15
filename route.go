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
		node       = &t.root
		paramNames []string
	)
	for _, part := range parts {
		switch part[0] {
		case colon:
			paramNames = append(paramNames, part[1:])
			if node.param == nil {
				node.param = &routeNode{}
			}

			node = node.param
		case '*':
			// The existing matcher ignores everything following a wildcard.
			if node.wildcard == nil {
				node.wildcard = &routeEndpoint{handler: handler, paramNames: paramNames}
			}

			return
		default:
			node = node.insertStatic(part)
		}
	}

	if node.endpoint == nil {
		node.endpoint = &routeEndpoint{handler: handler, paramNames: paramNames}
	}
}

// find appends captured parameters to borrowed storage. A miss restores its
// original length; successful captures get the selected endpoint's names.
func (t *route) find(url string, params *Params) *routeEndpoint {
	if t.count == 0 {
		return nil
	}

	var (
		start    = len(*params)
		endpoint = t.root.find(url, params)
	)
	if endpoint == nil {
		return nil
	}

	for index, name := range endpoint.paramNames {
		(*params)[start+index].Key = name
	}

	return endpoint
}
