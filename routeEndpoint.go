package httpserve

func newRouteEndpoint(handler Handler, parts []string, numParams int) *routeEndpoint {
	var endpoint = &routeEndpoint{handler: handler}
	if numParams == 0 {
		return endpoint
	}

	endpoint.paramNames = make([]string, 0, numParams)
	for _, part := range parts {
		if part[0] == colon {
			endpoint.paramNames = append(endpoint.paramNames, part[1:])
			if len(endpoint.paramNames) == numParams {
				break
			}
		}
	}

	return endpoint
}

// routeEndpoint holds the handler and parameter names for a terminal trie match.
// Values are filled as traversal returns; names belong to the selected endpoint.
type routeEndpoint struct {
	handler    Handler
	paramNames []string
}

// prepareParams reserves exactly enough storage for the winning route while
// preserving any caller-supplied prefix. A nil endpoint represents a failed branch.
func (e *routeEndpoint) prepareParams(params *Params, length int) *routeEndpoint {
	if e == nil {
		return nil
	}

	if cap(*params) < length {
		var grown = make(Params, length)
		copy(grown, *params)
		*params = grown
	} else {
		*params = (*params)[:length]
	}

	return e
}
