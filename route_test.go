package httpserve

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestRouterTrieMatching(t *testing.T) {
	var tests = []struct {
		name      string
		patterns  []string
		path      string
		wantRoute int
		want      Params
	}{
		{name: "root", patterns: []string{"/"}, path: "/", wantRoute: 0},
		{name: "empty_url", patterns: []string{"/", "/:id", "/*"}, path: "", wantRoute: -1},
		{name: "one_byte_static_fallback", patterns: []string{"/a/x", "/a/y", "/:first/:second"}, path: "/a/z", wantRoute: 2, want: Params{{Key: "first", Value: "a"}, {Key: "second", Value: "z"}}},
		{name: "one_byte_wildcard_boundary", patterns: []string{"/a/*", "/ab/end", "/b"}, path: "/ab/missing", wantRoute: -1},
		{name: "exact", patterns: []string{"/api/items"}, path: "/api/items", wantRoute: 0},
		{name: "prefix_is_not_match", patterns: []string{"/api/items"}, path: "/api/itemssuffix", wantRoute: -1},
		{name: "trailing_slash_is_not_redirected", patterns: []string{"/api/items"}, path: "/api/items/", wantRoute: -1},
		{name: "registration_slashes_normalized", patterns: []string{"/api//items/"}, path: "/api/items", wantRoute: 0},
		{name: "request_slashes_not_normalized", patterns: []string{"/api//items/"}, path: "/api//items/", wantRoute: -1},
		{name: "split_static_edge", patterns: []string{"/a/alpha", "/a/alpine", "/a/al", "/a", "/a/alpha/deeper"}, path: "/a/alpine", wantRoute: 1},
		{name: "shorter_static_edge", patterns: []string{"/a/alpha", "/a/alpine", "/a/al", "/a", "/a/alpha/deeper"}, path: "/a/al", wantRoute: 2},
		{name: "extend_static_edge", patterns: []string{"/a", "/a/alpha", "/a/alpha/deeper"}, path: "/a/alpha/deeper", wantRoute: 2},
		{name: "empty_parameter", patterns: []string{"/:id"}, path: "/", wantRoute: 0, want: Params{{Key: "id", Value: ""}}},
		{name: "adjacent_parameters", patterns: []string{"/:a/:b/:c"}, path: "/one/two/three", wantRoute: 0, want: Params{{Key: "a", Value: "one"}, {Key: "b", Value: "two"}, {Key: "c", Value: "three"}}},
		{name: "parameter_before_static_suffix", patterns: []string{"/api/:platform/:environment/latest"}, path: "/api/win32/staging/latest", wantRoute: 0, want: Params{{Key: "platform", Value: "win32"}, {Key: "environment", Value: "staging"}}},
		{name: "wildcard", patterns: []string{"/files/*"}, path: "/files/assets/icon.svg", wantRoute: 0},
		{name: "wildcard_requires_remainder", patterns: []string{"/files/*"}, path: "/files", wantRoute: -1},
		{name: "wildcard_slash_remainder", patterns: []string{"/files/*"}, path: "/files/", wantRoute: 0},
		{name: "wildcard_boundary", patterns: []string{"/files/*", "/files-other/item"}, path: "/files-other/missing", wantRoute: -1},
		{name: "wildcard_preserves_ignored_tail", patterns: []string{"/files/*/ignored/:id"}, path: "/files/x", wantRoute: 0},
		{name: "parameter_then_wildcard", patterns: []string{"/users/:id/*"}, path: "/users/42/assets/icon.svg", wantRoute: 0, want: Params{{Key: "id", Value: "42"}}},
		{name: "static_beats_parameter", patterns: []string{"/:id", "/users"}, path: "/users", wantRoute: 1},
		{name: "static_beats_parameter_reversed", patterns: []string{"/users", "/:id"}, path: "/users", wantRoute: 0},
		{name: "parameter_beats_wildcard", patterns: []string{"/files/*", "/files/:id"}, path: "/files/icon", wantRoute: 1, want: Params{{Key: "id", Value: "icon"}}},
		{name: "parameter_beats_wildcard_reversed", patterns: []string{"/files/:id", "/files/*"}, path: "/files/icon", wantRoute: 0, want: Params{{Key: "id", Value: "icon"}}},
		{name: "static_branch_beats_parameter_suffix", patterns: []string{"/files/:id/detail", "/files/special/*"}, path: "/files/special/detail", wantRoute: 1},
		{name: "static_falls_back_to_parameter", patterns: []string{"/users/info", "/:id/settings"}, path: "/users/settings", wantRoute: 1, want: Params{{Key: "id", Value: "users"}}},
		{name: "failed_branch_parameters_discarded", patterns: []string{"/users/:first/details", "/:second/:third"}, path: "/users/alice", wantRoute: 1, want: Params{{Key: "second", Value: "users"}, {Key: "third", Value: "alice"}}},
		{name: "parameter_falls_back_to_wildcard", patterns: []string{"/users/:id/details", "/users/*"}, path: "/users/alice/other", wantRoute: 1},
		{name: "wildcard_fallback_keeps_prefix_parameter", patterns: []string{"/files/:owner/:id/details", "/files/:user/*"}, path: "/files/alice/42/other", wantRoute: 1, want: Params{{Key: "user", Value: "alice"}}},
		{name: "wildcard_ignores_later_parameter", patterns: []string{"/users/:id/*/:unused"}, path: "/users/42/assets", wantRoute: 0, want: Params{{Key: "id", Value: "42"}}},
		{name: "failed_match_clears_parameters", patterns: []string{"/users/:id/details"}, path: "/users/alice/other", wantRoute: -1},
		{name: "shared_branch_keeps_parameter_names", patterns: []string{"/:first/details", "/:second/settings"}, path: "/alice/settings", wantRoute: 1, want: Params{{Key: "second", Value: "alice"}}},
		{name: "duplicate_keeps_first_handler", patterns: []string{"/users", "/users"}, path: "/users", wantRoute: 0},
		{name: "equivalent_parameters_keep_first_names", patterns: []string{"/users/:id", "/users/:name"}, path: "/users/alice", wantRoute: 0, want: Params{{Key: "id", Value: "alice"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				router   = newRouter()
				selected = -2
				captured Params
			)
			router.SetNotFound(func(ctx *Context) {
				selected = -1
				captured = append(Params(nil), ctx.Params...)
				ctx.Writer().WriteHeader(http.StatusNotFound)
			})
			for index, pattern := range test.patterns {
				if err := router.GET(pattern, func(ctx *Context) {
					selected = index
					captured = append(Params(nil), ctx.Params...)
					ctx.WriteNoContent()
				}); err != nil {
					t.Fatal(err)
				}
			}

			var handler, params, ok = router.Match(http.MethodGet, test.path)
			if ok != (test.wantRoute >= 0) || !slices.Equal(params, test.want) {
				t.Fatalf("Match = %t, %v; want %t, %v", ok, params, test.wantRoute >= 0, test.want)
			}

			var reused = make(Params, 1, 8)
			reused[0] = Param{Key: "previous", Value: "discarded"}
			var reusedHandler, reusedParams, reusedOK = router.MatchInto(http.MethodGet, test.path, reused)
			if reusedOK != ok || !slices.Equal(reusedParams, params) {
				t.Fatalf("MatchInto = %t/%v, want %t/%v", reusedOK, reusedParams, ok, params)
			}

			// Set Path directly because NewRequest rejects an empty request target.
			var request = httptest.NewRequest(http.MethodGet, "/", nil)
			request.URL.Path = test.path
			reusedHandler(newContext(httptest.NewRecorder(), request, reusedParams))
			if selected != test.wantRoute {
				t.Fatalf("MatchInto selected route %d, want %d", selected, test.wantRoute)
			}

			handler(newContext(httptest.NewRecorder(), request, params))
			if selected != test.wantRoute {
				t.Fatalf("Match selected route %d, want %d", selected, test.wantRoute)
			}

			selected = -2
			router.ServeHTTP(httptest.NewRecorder(), request)
			if selected != test.wantRoute || !slices.Equal(captured, test.want) {
				t.Fatalf("ServeHTTP selected route/params %d/%v, want %d/%v", selected, captured, test.wantRoute, test.want)
			}
		})
	}
}

func TestRouterTrieParameterStorage(t *testing.T) {
	var (
		router = newRouter()
		want   = Params{{Key: "first", Value: "a"}, {Key: "second", Value: "b"}, {Key: "third", Value: "c"}, {Key: "fourth", Value: "d"}, {Key: "fifth", Value: "e"}}
	)
	// The static branch fails after five parameters. Only the successful fallback
	// may grow storage, and it must preserve the prefix and supply its own names.
	if err := router.GET("/users/:a/:b/:c/:d/:e/details", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	if err := router.GET("/:first/:second/:third/:fourth/:fifth/:sixth", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	var (
		params = Params{{Key: "existing", Value: "kept"}}
		prefix = params[0]
		path   = "/users/a/b/c/d/e"
	)
	router.match(http.MethodGet, path, &params)
	var fallback = Params{{Key: "first", Value: "users"}, {Key: "second", Value: "a"}, {Key: "third", Value: "b"}, {Key: "fourth", Value: "c"}, {Key: "fifth", Value: "d"}, {Key: "sixth", Value: "e"}}
	if params[0] != prefix || !slices.Equal(params[1:], fallback) {
		t.Fatalf("borrowed params = %v, want prefix %v followed by %v", params, prefix, fallback)
	}

	if err := router.GET("/owned/:first/:second/:third/:fourth/:fifth", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	var _, owned, ok = router.Match(http.MethodGet, "/owned/a/b/c/d/e")
	if !ok || !slices.Equal(owned, want) {
		t.Fatalf("owned params = %t/%v, want true/%v", ok, owned, want)
	}

	router.Match(http.MethodGet, "/owned/1/2/3/4/5")
	if !slices.Equal(owned, want) {
		t.Fatalf("subsequent match changed owned params to %v", owned)
	}

	params = params[:0]
	router.match(http.MethodGet, "/users/a/b/c/d/e/missing/extra", &params)
	if len(params) != 0 {
		t.Fatalf("miss retained parameters: %v", params)
	}
}

func TestRouterHandleInvalidPath(t *testing.T) {
	var router = newRouter()
	for _, pattern := range []string{"", "users/:id"} {
		if err := router.GET(pattern, func(ctx *Context) {}); err == nil || !strings.Contains(err.Error(), ErrMissingLeadSlash.Error()) {
			t.Fatalf("register %q returned %v, want missing-leading-slash error", pattern, err)
		}
	}

	if _, _, ok := router.Match(http.MethodGet, ""); ok {
		t.Fatal("invalid registration added an empty-path endpoint")
	}
}

func TestRouterTrieMethods(t *testing.T) {
	var (
		router   = newRouter()
		methods  = []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions}
		selected string
	)
	router.SetNotFound(func(ctx *Context) { selected = "not-found" })
	for _, method := range methods {
		if err := router.Handle(method, "/items/:id", func(ctx *Context) { selected = method }); err != nil {
			t.Fatal(err)
		}
	}

	for _, method := range append(methods, http.MethodPatch) {
		t.Run(method, func(t *testing.T) {
			var (
				want    = method
				wantOK  = method != http.MethodPatch
				request = httptest.NewRequest(method, "/items/42", nil)
			)
			if !wantOK {
				want = "not-found"
			}

			var handler, params, ok = router.Match(method, request.URL.Path)
			if ok != wantOK {
				t.Fatalf("Match = %t, want %t", ok, wantOK)
			}

			selected = ""
			handler(newContext(httptest.NewRecorder(), request, params))
			if selected != want {
				t.Fatalf("Match selected %q, want %q", selected, want)
			}

			selected = ""
			router.ServeHTTP(httptest.NewRecorder(), request)
			if selected != want {
				t.Fatalf("ServeHTTP selected %q, want %q", selected, want)
			}
		})
	}

	if err := router.Handle(http.MethodPatch, "/items", func(ctx *Context) {}); err == nil {
		t.Fatal("unsupported method registration succeeded")
	}
}

func TestRouterTrieConcurrentRequests(t *testing.T) {
	var router = newRouter()
	if err := router.GET("/users/:id", func(ctx *Context) {
		ctx.WriteString(http.StatusOK, "text/plain", ctx.Param("id"))
	}); err != nil {
		t.Fatal(err)
	}

	if err := router.GET("/users/current", func(ctx *Context) {
		ctx.WriteString(http.StatusOK, "text/plain", "current")
	}); err != nil {
		t.Fatal(err)
	}

	var workers sync.WaitGroup
	for index := 0; index < 20; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			var value = strconv.Itoa(index)
			if index%2 == 0 {
				value = "current"
			}

			var request = httptest.NewRequest(http.MethodGet, "/users/"+value, nil)
			for iteration := 0; iteration < 100; iteration++ {
				var writer = httptest.NewRecorder()
				router.ServeHTTP(writer, request)
				if writer.Code != http.StatusOK || writer.Body.String() != value {
					t.Errorf("response = %d/%q, want 200/%q", writer.Code, writer.Body.String(), value)
					return
				}
			}
		}()
	}

	workers.Wait()
}

// FuzzRouterTrie checks match existence against the original route checker.
// Overlapping routes may have a new precedence, but the selected route must match
// and its parameter names and values must agree with that route's checker.
func FuzzRouterTrie(f *testing.F) {
	var seeds = [][2]string{
		{"/", "/"},
		{"/:id", "/"},
		{"/:id", "abc"},
		{"/*", "abc"},
		{"//", ""},
		{"/a/alpha\n/a/alpine\n/a/al", "/a/al"},
		{"/users/:first/details\n/:second/:third", "/users/alice"},
		{"/files/*\n/files-other/item", "/files-other/missing"},
		{"/users/:id/*", "/users/42/"},
		{"/files/*/ignored/:id", "/files/x"},
		{"/:a/:b/:c", "///"},
		{"/api//items/", "/api/items"},
		{"/api/:id/details", "/api/42/missing"},
	}
	for _, seed := range seeds {
		f.Add(seed[0], seed[1])
	}

	f.Fuzz(func(t *testing.T, patterns, path string) {
		if len(patterns) > 1024 || len(path) > 2048 {
			t.Skip()
		}

		var (
			router    = newRouter()
			reference []*referenceRoute
			selected  = -2
			wantOK    bool
		)
		router.SetNotFound(func(ctx *Context) { selected = -1 })
		for _, pattern := range strings.Split(patterns, "\n") {
			if len(pattern) == 0 || pattern[0] != '/' {
				continue
			}

			var index = len(reference)
			if err := router.GET(pattern, func(ctx *Context) { selected = index }); err != nil {
				t.Fatal(err)
			}

			var compiled, err = newReferenceRoute(pattern)
			if err != nil {
				t.Fatal(err)
			}

			reference = append(reference, compiled)
			if _, matches := compiled.check(nil, path); matches {
				wantOK = true
			}
		}

		var handler, params, ok = router.Match(http.MethodGet, path)
		if ok != wantOK {
			t.Fatalf("Match(%q) = %t, want %t for %q", path, ok, wantOK, patterns)
		}

		handler(newContext(nil, nil, params))
		if ok {
			if selected < 0 || selected >= len(reference) {
				t.Fatalf("selected route %d outside registered routes", selected)
			}

			var want, matches = reference[selected].check(nil, path)
			if !matches || !slices.Equal(params, want) {
				t.Fatalf("selected route %d with params %v; checker returned %t/%v", selected, params, matches, want)
			}
		} else if selected != -1 || len(params) != 0 {
			t.Fatalf("miss selected route/params %d/%v, want -1/[]", selected, params)
		}

		var (
			wantRoute = selected
			pooled    = make(Params, 0, 4)
		)
		selected = -2
		handler = router.match(http.MethodGet, path, &pooled)
		handler(newContext(nil, nil, pooled))
		if selected != wantRoute || !slices.Equal(pooled, params) {
			t.Fatalf("pooled lookup selected %d/%v, want %d/%v", selected, pooled, wantRoute, params)
		}

		handler, pooled, ok = router.MatchInto(http.MethodGet, path, pooled)
		handler(newContext(nil, nil, pooled))
		if ok != wantOK || selected != wantRoute || !slices.Equal(pooled, params) {
			t.Fatalf("MatchInto selected %t/%d/%v, want %t/%d/%v", ok, selected, pooled, wantOK, wantRoute, params)
		}
	})
}

func TestRouteFindMissPreservesStorage(t *testing.T) {
	var router = newRouter()
	if err := router.GET("/deep/:a/:b/:c/:d/:e/end", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	var (
		backing = Params{{Key: "prefix", Value: "kept"}, {Key: "spare", Value: "untouched"}}
		params  = backing[:1]
		before  = slices.Clone(backing)
	)
	if endpoint := router.rm[methodGET].find("/deep/1/2/3/4/5/missing", &params); endpoint != nil {
		t.Fatal("unexpected match")
	}

	if len(params) != 1 || cap(params) != 2 || &params[0] != &backing[0] || !slices.Equal(backing, before) {
		t.Fatalf("miss changed borrowed storage: %v", params[:cap(params)])
	}
}

func TestRouteCheck(t *testing.T) {
	var r = newRouter()
	if err := r.GET(smallRoute, func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	_, params, ok := r.Match("GET", smallRouteNoParam)
	if !ok {
		t.Fatal("Match not ok when it should be")
	}

	if value := params.ByName("name"); value != "name" {
		t.Fatalf("Invalid value for key \"%s\", expected \"%s\" and received \"%s\"", "name", "name", value)
	}

	_, _, ok = r.Match("GET", "test")
	if ok {
		t.Fatal("Match ok when it should not be")
	}
}

func TestRouteAfterParam(t *testing.T) {
	var r = newRouter()
	if err := r.GET("/api/releases/hatch/:platform/:environment/latest", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	_, params, ok := r.Match("GET", "/api/releases/hatch/win32/staging/latest")
	if !ok {
		t.Fatal("Match not ok when it should be")
	}

	if value := params.ByName("platform"); value != "win32" {
		t.Fatalf("Invalid value for key \"%s\", expected \"%s\" and received \"%s\"", "platform", "win32", value)
	}
}
