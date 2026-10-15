package httpserve

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkRouterMatch measures the public lookup API, including owned Params.
func BenchmarkRouterMatch(b *testing.B) {
	runRouterBenchmarks(b, false)
}

// BenchmarkRouterServeHTTP measures the pooled lookup used to serve requests.
func BenchmarkRouterServeHTTP(b *testing.B) {
	runRouterBenchmarks(b, true)
}

func runRouterBenchmarks(b *testing.B, serveHTTP bool) {
	var lengths = []struct {
		name   string
		prefix string
	}{
		{name: "Short", prefix: ""},
		{name: "Medium", prefix: "/api/v1/projects/catalog/items"},
		{name: "Long", prefix: "/api/v1/organizations/example-corporation/workspaces/production/projects/product-catalog/environments/staging/regions/us-west/resources/collections/inventory/items"},
	}
	for _, length := range lengths {
		b.Run(length.name, func(b *testing.B) {
			for _, kind := range []string{"Static", "Param", "Wildcard"} {
				b.Run(kind, func(b *testing.B) {
					for _, routeCount := range []int{1, 100, 1000} {
						b.Run(fmt.Sprintf("Routes=%d", routeCount), func(b *testing.B) {
							var targets = []struct {
								name  string
								index int
							}{
								{name: "FirstHit", index: 0},
								{name: "LastHit", index: routeCount - 1},
								{name: "Miss", index: routeCount},
							}
							for _, target := range targets {
								b.Run(target.name, func(b *testing.B) {
									benchmarkRouterLookup(b, length.prefix, kind, routeCount, target.index, serveHTTP)
								})
							}
						})
					}
				})
			}
		})
	}
}

func benchmarkRouterLookup(b *testing.B, prefix, kind string, routeCount, target int, serveHTTP bool) {
	b.Helper()
	var (
		router       = newRouter()
		matchedRoute = -1
		writer       = &routerBenchmarkWriter{header: make(http.Header)}
		requestPath  = fmt.Sprintf("%s/r%04d/42", prefix, target)
		request      = httptest.NewRequest(http.MethodGet, requestPath, nil)
		wantMatch    = target < routeCount
		wantStatus   = http.StatusNoContent
		wantRoute    = target
	)
	if !wantMatch {
		wantStatus = http.StatusNotFound
		wantRoute = -1
	}

	router.SetNotFound(func(ctx *Context) {
		matchedRoute = -1
		ctx.Writer().WriteHeader(http.StatusNotFound)
	})
	for index := 0; index < routeCount; index++ {
		var pattern = fmt.Sprintf("%s/r%04d", prefix, index)
		switch kind {
		case "Static":
			pattern += "/42"
		case "Param":
			pattern += "/:id"
		case "Wildcard":
			pattern += "/*"
		}

		if err := router.GET(pattern, func(ctx *Context) {
			matchedRoute = index
			ctx.Writer().WriteHeader(http.StatusNoContent)
		}); err != nil {
			b.Fatalf("register %q: %v", pattern, err)
		}
	}

	// Check correctness before timing so a faster wrong answer cannot look good.
	var handler, params, ok = router.Match(http.MethodGet, requestPath)
	if ok != wantMatch {
		b.Fatalf("Match(%q) = %t, want %t", requestPath, ok, wantMatch)
	}

	var wantParams int
	if wantMatch && kind == "Param" {
		wantParams = 1
		if value := params.ByName("id"); value != "42" {
			b.Fatalf("id = %q, want 42", value)
		}
	}

	if len(params) != wantParams {
		b.Fatalf("parameter count = %d, want %d", len(params), wantParams)
	}

	handler(newContext(writer, request, params))
	if matchedRoute != wantRoute || writer.status != wantStatus {
		b.Fatalf("handler selected route/status %d/%d, want %d/%d", matchedRoute, writer.status, wantRoute, wantStatus)
	}

	b.ReportAllocs()
	if serveHTTP {
		// Warm the Context pool and verify the production lookup separately.
		matchedRoute = -2
		writer.status = 0
		router.ServeHTTP(writer, request)
		if matchedRoute != wantRoute || writer.status != wantStatus {
			b.Fatalf("ServeHTTP selected route/status %d/%d, want %d/%d", matchedRoute, writer.status, wantRoute, wantStatus)
		}

		matchedRoute = -2
		writer.status = 0
		b.ResetTimer()
		for iteration := 0; iteration < b.N; iteration++ {
			router.ServeHTTP(writer, request)
		}

		b.StopTimer()
		if matchedRoute != wantRoute || writer.status != wantStatus {
			b.Fatalf("ServeHTTP selected route/status %d/%d, want %d/%d", matchedRoute, writer.status, wantRoute, wantStatus)
		}

		return
	}

	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		handlerSink, paramsSink, boolSink = router.Match(http.MethodGet, requestPath)
	}

	b.StopTimer()
	if boolSink != wantMatch || len(paramsSink) != wantParams {
		b.Fatalf("timed Match returned match/parameter count %t/%d, want %t/%d", boolSink, len(paramsSink), wantMatch, wantParams)
	}
}
