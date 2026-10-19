package apex

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkRouterByteSearch exercises delimiter scans that the short IDs in the
// existing router suite do not cover. All requests and routes are built untimed.
func BenchmarkRouterByteSearch(b *testing.B) {
	for _, length := range []int{2, 8, 16, 32, 64, 256, 1024} {
		for _, suffix := range []string{"", "/details"} {
			var ending = "Terminal"
			if suffix != "" {
				ending = "Nested"
			}

			b.Run(fmt.Sprintf("Bytes=%d/%s", length, ending), func(b *testing.B) {
				var (
					router     = newRouter()
					id         = strings.Repeat("x", length)
					request    = httptest.NewRequest(http.MethodGet, "/items/"+id+suffix, nil)
					writer     = &routerBenchmarkWriter{header: make(http.Header)}
					receivedID string
				)
				if err := router.GET("/items/:id"+suffix, func(ctx *Context) {
					receivedID = ctx.Param("id")
					ctx.Writer().WriteHeader(http.StatusNoContent)
				}); err != nil {
					b.Fatal(err)
				}

				router.ServeHTTP(writer, request)
				if receivedID != id || writer.status != http.StatusNoContent {
					b.Fatalf("dispatch = %q/%d, want %q/204", receivedID, writer.status, id)
				}

				receivedID = ""
				writer.status = 0
				b.ReportAllocs()
				b.ResetTimer()
				for iteration := 0; iteration < b.N; iteration++ {
					router.ServeHTTP(writer, request)
				}

				b.StopTimer()
				if receivedID != id || writer.status != http.StatusNoContent {
					b.Fatalf("timed dispatch = %q/%d, want %q/204", receivedID, writer.status, id)
				}
			})
		}
	}
}

// BenchmarkRouterFanout rotates through sibling edges so branch prediction is
// not trained exclusively on a single successful URL.
func BenchmarkRouterFanout(b *testing.B) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	for _, count := range []int{4, 16, 64} {
		b.Run(fmt.Sprintf("Children=%d", count), func(b *testing.B) {
			var (
				router   = newRouter()
				requests = make([]*http.Request, count)
				writer   = &routerBenchmarkWriter{header: make(http.Header)}
				selected = -1
			)
			for index := range requests {
				var path = "/" + alphabet[index:index+1] + "/details"
				requests[index] = httptest.NewRequest(http.MethodGet, path, nil)
				if err := router.GET(path, func(ctx *Context) {
					selected = index
					ctx.Writer().WriteHeader(http.StatusNoContent)
				}); err != nil {
					b.Fatal(err)
				}
			}

			for index, request := range requests {
				router.ServeHTTP(writer, request)
				if selected != index || writer.status != http.StatusNoContent {
					b.Fatalf("dispatch = %d/%d, want %d/204", selected, writer.status, index)
				}
			}

			selected = -1
			writer.status = 0
			b.ReportAllocs()
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				router.ServeHTTP(writer, requests[iteration&(count-1)])
			}

			b.StopTimer()
			if selected != (b.N-1)&(count-1) || writer.status != http.StatusNoContent {
				b.Fatalf("timed dispatch = %d/%d, want %d/204", selected, writer.status, (b.N-1)&(count-1))
			}
		})
	}
}
