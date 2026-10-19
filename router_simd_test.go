package apex

import (
	"net/http"
	"strings"
	"testing"
)

func TestRouterSIMDParameterBoundaries(t *testing.T) {
	var router = newRouter()
	if err := router.GET("/items/:id/details", func(ctx *Context) {}); err != nil {
		t.Fatal(err)
	}

	// Include the size of /details in the search. This crosses the 32-byte
	// cutover and tests empty captures, NUL, and non-ASCII bytes as well.
	for _, target := range []byte{'x', 0, 0x81, 0xff} {
		for length := 0; length <= 65; length++ {
			var value = strings.Repeat(string([]byte{target}), length)
			var _, params, ok = router.Match(http.MethodGet, "/items/"+value+"/details")
			if !ok || len(params) != 1 || params.ByName("id") != value {
				t.Fatalf("byte=%x length=%d: matched %t with %v", target, length, ok, params)
			}
		}
	}
}

func TestRouterSIMDChildBoundaries(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	for _, count := range []int{15, 16, 31, 32, 33, 64} {
		var (
			router   = newRouter()
			selected = -1
		)
		for index := 0; index < count; index++ {
			var path = "/" + alphabet[index:index+1] + "/details"
			if err := router.GET(path, func(ctx *Context) { selected = index }); err != nil {
				t.Fatal(err)
			}
		}

		for index := 0; index < count; index++ {
			var path = "/" + alphabet[index:index+1] + "/details"
			var handler, params, ok = router.Match(http.MethodGet, path)
			if !ok || len(params) != 0 {
				t.Fatalf("children=%d path=%q: matched %t with %v", count, path, ok, params)
			}

			selected = -1
			handler(nil)
			if selected != index {
				t.Fatalf("children=%d path=%q: selected %d, want %d", count, path, selected, index)
			}
		}

		if _, _, ok := router.Match(http.MethodGet, "/~/details"); ok {
			t.Fatalf("children=%d: unexpected match", count)
		}
	}
}
