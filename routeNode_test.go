package apex

import (
	"slices"
	"testing"
)

func TestRouteNodePrefixMatching(t *testing.T) {
	var tests = []struct {
		name   string
		prefix string
		url    string
		match  bool
	}{
		{name: "empty_prefix_and_url", match: true},
		{name: "empty_prefix_with_remainder", url: "x"},
		{name: "one_byte_empty_url", prefix: "/"},
		{name: "one_byte_mismatch", prefix: "/", url: "x"},
		{name: "one_byte_match", prefix: "/", url: "/", match: true},
		{name: "one_byte_with_remainder", prefix: "/", url: "/x"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				endpoint = &routeEndpoint{}
				node     = routeNode{prefix: test.prefix, endpoint: endpoint}
				backing  = Params{{Key: "prefix", Value: "kept"}, {Key: "spare", Value: "untouched"}}
				params   = backing[:1]
				before   = slices.Clone(backing)
				want     *routeEndpoint
			)
			if test.match {
				want = endpoint
			}

			if got := node.find(test.url, &params, len(params)); got != want {
				t.Fatalf("find(%q) = %p, want %p", test.url, got, want)
			}

			if len(params) != 1 || cap(params) != 2 || &params[0] != &backing[0] || !slices.Equal(backing, before) {
				t.Fatalf("prefix comparison changed capture storage: %v", params[:cap(params)])
			}
		})
	}
}
