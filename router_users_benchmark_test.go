package httpserve

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkRouterUsers measures the common 19-byte URL /api/users/00000000
// against small API route tables. User IDs share the /api/users/:id pattern.
func BenchmarkRouterUsers(b *testing.B) {
	for _, routeCount := range []int{20, 40, 50, 100} {
		b.Run(fmt.Sprintf("Routes=%d", routeCount), func(b *testing.B) {
			var cases = []struct {
				name     string
				position int
				miss     bool
			}{
				{name: "FirstHit", position: 0},
				{name: "MiddleHit", position: routeCount / 2},
				{name: "LastHit", position: routeCount - 1},
				{name: "Miss", position: routeCount - 1, miss: true},
			}
			for _, lookup := range cases {
				b.Run(lookup.name, func(b *testing.B) {
					benchmarkRouterUsers(b, routeCount, lookup.position, lookup.miss)
				})
			}
		})
	}
}

func benchmarkRouterUsers(b *testing.B, routeCount, position int, miss bool) {
	b.Helper()
	var (
		router       = newRouter()
		writer       = &routerBenchmarkWriter{header: make(http.Header)}
		requestPath  = "/api/users/00000000"
		wantRoute    = position
		wantID       = "00000000"
		wantStatus   = http.StatusNoContent
		wantParams   = 1
		matchedRoute = -2
		receivedID   string
	)
	if miss {
		requestPath = "/api/other/00000000"
		wantRoute = -1
		wantID = ""
		wantStatus = http.StatusNotFound
		wantParams = 0
	}

	router.SetNotFound(func(ctx *Context) {
		matchedRoute = -1
		receivedID = ctx.Param("id")
		ctx.Writer().WriteHeader(http.StatusNotFound)
	})
	for index := 0; index < routeCount; index++ {
		var pattern = fmt.Sprintf("/api/r%04d/:id", index)
		if index == position {
			pattern = "/api/users/:id"
		}

		if err := router.GET(pattern, func(ctx *Context) {
			matchedRoute = index
			receivedID = ctx.Param("id")
			ctx.Writer().WriteHeader(http.StatusNoContent)
		}); err != nil {
			b.Fatalf("register %q: %v", pattern, err)
		}
	}

	var _, params, ok = router.Match(http.MethodGet, requestPath)
	if ok == miss || len(params) != wantParams || params.ByName("id") != wantID {
		b.Fatalf("Match(%q) = %t/%v, want match=%t, id=%q", requestPath, ok, params, !miss, wantID)
	}

	var request = httptest.NewRequest(http.MethodGet, requestPath, nil)
	router.ServeHTTP(writer, request)
	if matchedRoute != wantRoute || receivedID != wantID || writer.status != wantStatus {
		b.Fatalf("ServeHTTP = route/id/status %d/%q/%d, want %d/%q/%d", matchedRoute, receivedID, writer.status, wantRoute, wantID, wantStatus)
	}

	matchedRoute = -2
	receivedID = "unset"
	writer.status = 0
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		router.ServeHTTP(writer, request)
	}

	b.StopTimer()
	if matchedRoute != wantRoute || receivedID != wantID || writer.status != wantStatus {
		b.Fatalf("timed ServeHTTP = route/id/status %d/%q/%d, want %d/%q/%d", matchedRoute, receivedID, writer.status, wantRoute, wantID, wantStatus)
	}
}
