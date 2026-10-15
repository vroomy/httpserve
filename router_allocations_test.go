package httpserve

import (
	"net/http"
	"slices"
	"testing"
)

func TestRouterMatchAllocations(t *testing.T) {
	var router = newRouter()
	for _, pattern := range []string{
		"/health", "/users/:id", "/files/*/:ignored/:alsoIgnored",
		"/deep/:a/:b/:c/:d/:e/end", "/fallback/:a/:b/:c/:d/:e/end", "/fallback/*",
	} {
		if err := router.GET(pattern, func(ctx *Context) {}); err != nil {
			t.Fatal(err)
		}
	}

	if err := router.POST("/:a/:b/:c/:d/:e/:f", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	var cases = []struct {
		name   string
		method string
		path   string
		match  bool
		params int
	}{
		{name: "static", method: http.MethodGet, path: "/health", match: true},
		{name: "one_parameter", method: http.MethodGet, path: "/users/42", match: true, params: 1},
		{name: "wildcard_ignored_tail", method: http.MethodGet, path: "/files/icon.svg", match: true},
		{name: "early_miss", method: http.MethodGet, path: "/missing"},
		{name: "empty_method", method: http.MethodHead, path: "/health"},
		{name: "unknown_method", method: http.MethodPatch, path: "/health"},
		{name: "deep_hit", method: http.MethodGet, path: "/deep/1/2/3/4/5/end", match: true, params: 5},
		{name: "deep_miss", method: http.MethodGet, path: "/deep/1/2/3/4/5/missing"},
		{name: "fallback", method: http.MethodGet, path: "/fallback/1/2/3/4/5/missing", match: true},
	}
	for _, lookup := range cases {
		t.Run(lookup.name, func(t *testing.T) {
			var wantAllocs float64
			if lookup.params > 0 {
				wantAllocs = 1
			}

			var allocs = testing.AllocsPerRun(100, func() {
				handlerSink, paramsSink, boolSink = router.Match(lookup.method, lookup.path)
			})
			if allocs != wantAllocs || boolSink != lookup.match || len(paramsSink) != lookup.params || cap(paramsSink) != lookup.params {
				t.Fatalf("Match = %v allocs, %t, len/cap %d/%d; want %v, %t, %d/%d", allocs, boolSink, len(paramsSink), cap(paramsSink), wantAllocs, lookup.match, lookup.params, lookup.params)
			}

			var params = make(Params, 0, 5)
			allocs = testing.AllocsPerRun(100, func() {
				handlerSink, params, boolSink = router.MatchInto(lookup.method, lookup.path, params)
			})
			if allocs != 0 || boolSink != lookup.match || len(params) != lookup.params {
				t.Fatalf("MatchInto = %v allocs, %t, %v", allocs, boolSink, params)
			}
		})
	}
}

func TestRouterMatchIntoReuse(t *testing.T) {
	var router = newRouter()
	for _, pattern := range []string{"/:a/:b/:c/:d/:e", "/users/:id", "/files/*"} {
		if err := router.GET(pattern, func(ctx *Context) {}); err != nil {
			t.Fatal(err)
		}
	}

	var _, params, ok = router.MatchInto(http.MethodGet, "/1/2/3/4/5", nil)
	if !ok || len(params) != 5 {
		t.Fatalf("initial match = %t/%v", ok, params)
	}

	var backing = params[:cap(params)]
	_, params, ok = router.MatchInto(http.MethodGet, "/users/42", params)
	if !ok || !slices.Equal(params, Params{{Key: "id", Value: "42"}}) || &params[0] != &backing[0] {
		t.Fatalf("reuse = %t/%v", ok, params)
	}

	for _, param := range backing[1:] {
		if param != (Param{}) {
			t.Fatalf("reuse retained previous capture: %v", param)
		}
	}

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPatch} {
		backing[0] = Param{Key: "old", Value: "discarded"}
		_, params, ok = router.MatchInto(method, "/missing", backing[:1])
		if ok || len(params) != 0 || cap(params) != cap(backing) || backing[0] != (Param{}) {
			t.Fatalf("%s miss = %t/%v, backing = %v", method, ok, params, backing)
		}
	}
}
