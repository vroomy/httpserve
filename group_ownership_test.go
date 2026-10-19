package apex

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestGroupHandlerOwnership(t *testing.T) {
	for _, capacity := range []int{3, 4} {
		t.Run(fmt.Sprintf("capacity_%d", capacity), func(t *testing.T) {
			server := New()
			handlers := make([]Handler, 3, capacity)
			for i := range handlers {
				handlers[i] = func(ctx *Context) { ctx.Put("order", ctx.Get("order")+"P") }
			}

			parent := server.Group("/parent", handlers...)
			groups := map[string]Group{"parent": parent}
			for _, child := range []string{"a", "b"} {
				label := child
				groups[child] = parent.Group("/"+child, func(ctx *Context) { ctx.Put("order", ctx.Get("order")+label) })
			}

			for _, group := range groups {
				for _, method := range []string{"GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD"} {
					for _, suffix := range []string{"first", "second"} {
						label := suffix
						register := func(route string, hs ...Handler) error { return group.Handle(method, route, hs...) }
						switch method {
						case "GET":
							register = group.GET
						case "PUT":
							register = group.PUT
						case "POST":
							register = group.POST
						case "DELETE":
							register = group.DELETE
						case "OPTIONS":
							register = group.OPTIONS
						}

						if err := register("/"+suffix, func(ctx *Context) { ctx.WriteString(200, "text/plain", ctx.Get("order")+label) }); err != nil {
							t.Fatal(err)
						}
					}
				}
			}

			handlers[0] = func(ctx *Context) { ctx.Put("order", "overwritten") }
			for key := range groups {
				prefix, wantPrefix := "/parent", "PPP"
				if key != "parent" {
					prefix += "/" + key
					wantPrefix += key
				}

				for _, method := range []string{"GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD"} {
					for _, suffix := range []string{"first", "second"} {
						recorder := httptest.NewRecorder()
						server.g.r.ServeHTTP(recorder, httptest.NewRequest(method, prefix+"/"+suffix, nil))
						if recorder.Code != 200 || recorder.Body.String() != wantPrefix+suffix {
							t.Errorf("%s %s/%s = %d %q", method, prefix, suffix, recorder.Code, recorder.Body.String())
						}
					}
				}
			}
		})
	}
}

func TestRouteOwnsCallerHandlers(t *testing.T) {
	server := New()
	handlers := []Handler{func(ctx *Context) { ctx.WriteString(200, "text/plain", "original") }}
	if err := server.GET("/", handlers...); err != nil {
		t.Fatal(err)
	}

	handlers[0] = func(ctx *Context) { ctx.WriteString(200, "text/plain", "replacement") }
	recorder := httptest.NewRecorder()
	server.g.r.ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))
	if recorder.Body.String() != "original" {
		t.Fatalf("route retained borrowed handlers: %q", recorder.Body.String())
	}
}
