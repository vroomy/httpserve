package httpserve

import "strings"

// newReferenceRoute compiles a pattern for the independent linear matcher used
// by the trie fuzz tests. This reference is excluded from library builds.
func newReferenceRoute(url string) (*referenceRoute, error) {
	if len(url) == 0 || url[0] != '/' {
		return nil, ErrMissingLeadSlash
	}

	var parts, err = getParts(url)
	if err != nil {
		return nil, err
	}

	return &referenceRoute{parts: parts}, nil
}

type referenceRoute struct {
	parts []string
}

// check preserves the original route matcher's slash boundaries, parameter
// consumption, and wildcard behavior without calling the trie implementation.
func (r *referenceRoute) check(params Params, url string) (Params, bool) {
	for _, part := range r.parts {
		if len(url) == 0 {
			return params, false
		}

		switch part[0] {
		case colon:
			var value = url[1:]
			var end = strings.IndexByte(value, '/')
			if end < 0 {
				end = len(value)
			}

			params = append(params, Param{Key: part[1:], Value: value[:end]})
			url = url[end+1:]
		case '*':
			return params, true
		default:
			if !strings.HasPrefix(url, part) {
				return params, false
			}

			url = url[len(part):]
			if len(url) > 0 && url[0] != '/' {
				return params, false
			}
		}
	}

	return params, len(url) == 0
}
