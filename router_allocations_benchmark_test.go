package apex

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// BenchmarkRouterAllocations separates owned results, reusable capture storage,
// capture growth, and warmed HTTP dispatch using the same mixed route table.
func BenchmarkRouterAllocations(b *testing.B) {
	var cases = []struct {
		name   string
		method string
		path   string
		route  int
		params Params
	}{
		{name: "Static", method: http.MethodGet, path: "/health", route: 0},
		{name: "Param", method: http.MethodGet, path: "/users/42", route: 1, params: Params{{Key: "id", Value: "42"}}},
		{name: "Wildcard", method: http.MethodGet, path: "/files/icon.svg", route: 2},
		{name: "Miss", method: http.MethodGet, path: "/missing", route: -1},
		{name: "EmptyMethod", method: http.MethodHead, path: "/health", route: -1},
		{name: "UnknownMethod", method: http.MethodPatch, path: "/health", route: -1},
		{name: "DeepHit", method: http.MethodGet, path: "/deep/1/2/3/4/5/end", route: 3, params: Params{
			{Key: "a", Value: "1"}, {Key: "b", Value: "2"}, {Key: "c", Value: "3"}, {Key: "d", Value: "4"}, {Key: "e", Value: "5"},
		}},
		{name: "DeepMiss", method: http.MethodGet, path: "/deep/1/2/3/4/5/missing", route: -1},
		{name: "Fallback", method: http.MethodGet, path: "/fallback/1/2/3/4/5/missing", route: 5},
	}
	for _, lookup := range cases {
		b.Run(lookup.name, func(b *testing.B) {
			for _, mode := range []string{"Match", "MatchInto", "BorrowedWarm", "BorrowedCap4", "ServeHTTP"} {
				b.Run(mode, func(b *testing.B) {
					var (
						router   = newRouter()
						selected = -2
						writer   = &routerBenchmarkWriter{header: make(http.Header)}
						request  = httptest.NewRequest(lookup.method, lookup.path, nil)
						patterns = []string{
							"/health", "/users/:id", "/files/*", "/deep/:a/:b/:c/:d/:e/end",
							"/fallback/:a/:b/:c/:d/:e/end", "/fallback/*",
						}
					)
					router.SetNotFound(func(ctx *Context) { selected = -1 })
					for index, pattern := range patterns {
						if err := router.GET(pattern, func(ctx *Context) { selected = index }); err != nil {
							b.Fatal(err)
						}
					}

					var handler, params, ok = router.Match(lookup.method, lookup.path)
					handler(newContext(writer, request, params))
					if ok != (lookup.route >= 0) || selected != lookup.route || !slices.Equal(params, lookup.params) {
						b.Fatalf("Match = %t/%d/%v, want %t/%d/%v", ok, selected, params, lookup.route >= 0, lookup.route, lookup.params)
					}

					var (
						warmParams = make(Params, 0, 5)
						base       = make(Params, 0, 4)
					)
					// Verify the borrowed path and warm the HTTP Context pool.
					router.match(lookup.method, lookup.path, &warmParams)(newContext(writer, request, warmParams))
					if selected != lookup.route || !slices.Equal(warmParams, lookup.params) {
						b.Fatalf("borrowed match = %d/%v, want %d/%v", selected, warmParams, lookup.route, lookup.params)
					}

					router.ServeHTTP(writer, request)
					if selected != lookup.route {
						b.Fatalf("ServeHTTP selected %d, want %d", selected, lookup.route)
					}

					b.ReportAllocs()
					b.ResetTimer()
					switch mode {
					case "Match":
						for iteration := 0; iteration < b.N; iteration++ {
							handlerSink, paramsSink, boolSink = router.Match(lookup.method, lookup.path)
						}
					case "BorrowedWarm":
						for iteration := 0; iteration < b.N; iteration++ {
							warmParams = warmParams[:0]
							handlerSink = router.match(lookup.method, lookup.path, &warmParams)
						}
					case "MatchInto":
						for iteration := 0; iteration < b.N; iteration++ {
							handlerSink, warmParams, boolSink = router.MatchInto(lookup.method, lookup.path, warmParams)
						}
					case "BorrowedCap4":
						for iteration := 0; iteration < b.N; iteration++ {
							// Reuse the original four slots to isolate the cost of growth.
							params = base[:0]
							handlerSink = router.match(lookup.method, lookup.path, &params)
						}
					case "ServeHTTP":
						for iteration := 0; iteration < b.N; iteration++ {
							router.ServeHTTP(writer, request)
						}
					}

					b.StopTimer()
					switch mode {
					case "Match":
						params = paramsSink
					case "BorrowedWarm", "MatchInto":
						params = warmParams
					case "ServeHTTP":
						if selected != lookup.route {
							b.Fatalf("timed ServeHTTP selected %d, want %d", selected, lookup.route)
						}

						return
					}

					if !slices.Equal(params, lookup.params) {
						b.Fatalf("timed params = %v, want %v", params, lookup.params)
					}
				})
			}
		})
	}
}

// BenchmarkRouterRegistration measures a fresh 40-route table. Pattern formatting
// and handler creation are excluded; each iteration includes router construction.
func BenchmarkRouterRegistration(b *testing.B) {
	for _, kind := range []string{"Static", "Param"} {
		b.Run(kind, func(b *testing.B) {
			var (
				patterns [40]string
				handler  Handler = func(ctx *Context) {}
			)
			for index := range patterns {
				patterns[index] = fmt.Sprintf("/api/r%04d", index)
				if kind == "Param" {
					patterns[index] += "/:id"
				}
			}

			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				var router = newRouter()
				for _, pattern := range patterns {
					if err := router.GET(pattern, handler); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
