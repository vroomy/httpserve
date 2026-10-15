package httpserve

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkRouterStdlib compares successful GET dispatch and native parameter
// lookup for the trie and http.ServeMux using identical 40-route fixtures.
func BenchmarkRouterStdlib(b *testing.B) {
	var urls = []struct {
		name     string
		prefix   string
		resource string
		id       string
	}{
		{name: "Common", prefix: "/api", resource: "users", id: "00000000"},
		{name: "Short", resource: "r0020", id: "42"},
		{name: "Medium", prefix: "/api/v1/projects/catalog/items", resource: "r0020", id: "42"},
		{name: "Long", prefix: "/api/v1/organizations/example-corporation/workspaces/production/projects/product-catalog/environments/staging/regions/us-west/resources/collections/inventory/items", resource: "r0020", id: "42"},
	}
	for _, url := range urls {
		b.Run("URL="+url.name, func(b *testing.B) {
			for _, implementation := range []string{"ServeMux", "Trie"} {
				b.Run("Router="+implementation, func(b *testing.B) {
					benchmarkRouterStdlib(b, implementation, url.prefix, url.resource, url.id)
				})
			}
		})
	}
}

func benchmarkRouterStdlib(b *testing.B, implementation, prefix, resource, id string) {
	b.Helper()
	const (
		routeCount = 40
		position   = routeCount / 2
	)
	var (
		trie         *Router
		mux          *http.ServeMux
		dispatch     http.Handler
		writer       = &routerBenchmarkWriter{header: make(http.Header)}
		requestPath  = fmt.Sprintf("%s/%s/%s", prefix, resource, id)
		request      = httptest.NewRequest(http.MethodGet, requestPath, nil)
		matchedRoute = -1
		receivedID   string
	)
	switch implementation {
	case "ServeMux":
		mux = http.NewServeMux()
		dispatch = mux
	case "Trie":
		trie = newRouter()
		dispatch = trie
	default:
		b.Fatalf("unknown router %q", implementation)
	}

	for index := 0; index < routeCount; index++ {
		var routeResource = fmt.Sprintf("r%04d", index)
		if index == position {
			routeResource = resource
		}

		var path = prefix + "/" + routeResource
		if mux != nil {
			mux.HandleFunc("GET "+path+"/{id}", func(w http.ResponseWriter, r *http.Request) {
				matchedRoute = index
				receivedID = r.PathValue("id")
				w.WriteHeader(http.StatusNoContent)
			})
			continue
		}

		if err := trie.GET(path+"/:id", func(ctx *Context) {
			matchedRoute = index
			receivedID = ctx.Param("id")
			ctx.Writer().WriteHeader(http.StatusNoContent)
		}); err != nil {
			b.Fatalf("register %q: %v", path, err)
		}
	}

	// Warm dispatch and verify equivalent handler, ID, and status before timing.
	dispatch.ServeHTTP(writer, request)
	if matchedRoute != position || receivedID != id || writer.status != http.StatusNoContent {
		b.Fatalf("%s dispatch = route/id/status %d/%q/%d, want %d/%q/204", implementation, matchedRoute, receivedID, writer.status, position, id)
	}

	matchedRoute = -1
	receivedID = "unset"
	writer.status = 0
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		dispatch.ServeHTTP(writer, request)
	}

	b.StopTimer()
	if matchedRoute != position || receivedID != id || writer.status != http.StatusNoContent {
		b.Fatalf("timed %s dispatch = route/id/status %d/%q/%d, want %d/%q/204", implementation, matchedRoute, receivedID, writer.status, position, id)
	}
}
