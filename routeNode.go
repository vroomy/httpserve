package httpserve

import "strings"

// routeNode compresses each static edge into a string. indices maps each
// child's first byte to its position in children without allocating on lookup.
// Parameter names stay on endpoints because a shared branch can serve routes with
// different names.
type routeNode struct {
	prefix   string
	indices  string
	children []*routeNode
	param    *routeNode
	endpoint *routeEndpoint
	wildcard *routeEndpoint
}

func (n *routeNode) insertStatic(prefix string) *routeNode {
	var index = strings.IndexByte(n.indices, prefix[0])
	if index < 0 {
		var child = &routeNode{prefix: prefix}
		n.indices += prefix[:1]
		n.children = append(n.children, child)
		return child
	}

	var (
		child  = n.children[index]
		shared int
	)
	for shared < len(prefix) && shared < len(child.prefix) && prefix[shared] == child.prefix[shared] {
		shared++
	}

	if shared < len(child.prefix) {
		var branch = &routeNode{
			prefix:   child.prefix[:shared],
			indices:  child.prefix[shared : shared+1],
			children: []*routeNode{child},
		}
		child.prefix = child.prefix[shared:]
		n.children[index] = branch
		child = branch
	}

	if shared == len(prefix) {
		return child
	}

	return child.insertStatic(prefix[shared:])
}

func (n *routeNode) find(url string, params *Params) *routeEndpoint {
	if !strings.HasPrefix(url, n.prefix) {
		return nil
	}

	url = url[len(n.prefix):]
	if len(url) == 0 {
		return n.endpoint
	}

	// At the end of a static part, the original matcher requires a slash
	// boundary. Radix splits inside a static part do not impose that boundary
	// on static children; only parameter and wildcard continuations need it.
	if index := strings.IndexByte(n.indices, url[0]); index >= 0 {
		if endpoint := n.children[index].find(url, params); endpoint != nil {
			return endpoint
		}
	}

	if n.prefix != "" && url[0] != '/' {
		return nil
	}

	if n.param != nil {
		var (
			start = len(*params)
			end   = strings.IndexByte(url[1:], '/')
		)
		if end < 0 {
			end = len(url) - 1
		}

		*params = append(*params, Param{Value: url[1 : end+1]})
		if endpoint := n.param.find(url[end+1:], params); endpoint != nil {
			return endpoint
		}

		// A failed parameter branch must not contaminate a wildcard or a
		// less specific ancestor branch. Retain any grown backing storage.
		*params = (*params)[:start]
	}

	return n.wildcard
}
