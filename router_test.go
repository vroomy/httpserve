package httpserve

import (
	"sync"
	"testing"
)

var (
	handlerSink Handler
	paramsSink  Params

	boolSink bool
)

func TestRouter(t *testing.T) {
	var err error
	cc := make(chan string, 4)
	r := newRouter()
	if err = r.GET(rootRoute, func(ctx *Context) {
		cc <- "root"
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(smallRoute, func(ctx *Context) {
		cc <- "small"
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(mediumRoute, func(ctx *Context) {
		cc <- "medium"
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(largeRoute, func(ctx *Context) {
		cc <- "large"
	}); err != nil {
		t.Fatal(err)
	}

	fn, params, ok := r.Match("GET", rootRoute)
	if !ok {
		t.Fatal("expected match and none was found")
	}

	if len(params) > 0 {
		t.Fatalf("unexpected params length, expected <0> and found <%d>", len(params))
	}

	fn(nil)

	if val := <-cc; val != "root" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", "root", val)
	}

	fn, params, ok = r.Match("GET", smallRouteNoParam)
	if !ok {
		t.Fatal("expected match and none was found")
	}

	fn(nil)

	if len(params) > 0 {
		t.Fatalf("unexpected params length, expected <0> and received <%d>", len(params))
	}

	if val := <-cc; val != "small" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", val, "small")
	}

	fn, params, ok = r.Match("GET", mediumRouteNoParam)
	if !ok {
		t.Fatal("expected match and none was found")
	}

	fn(nil)

	if len(params) > 0 {
		t.Fatalf("unexpected params length, expected <0> and received <%d>", len(params))
	}

	if val := <-cc; val != "medium" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", val, "small")
	}

	fn, params, ok = r.Match("GET", largeRouteNoParam)
	if !ok {
		t.Fatal("expected match and none was found")
	}

	fn(nil)

	if val := <-cc; val != "large" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", val, "small")
	}

	if params.ByName("name") != "name" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", "name", params.ByName("name"))
	}
}

func TestRouter_multi_level_handlers(t *testing.T) {
	r := newRouter()
	var wg sync.WaitGroup
	wg.Add(4)
	fn1 := func(ctx *Context) {
		wg.Done()
	}
	fn2 := func(ctx *Context) {
		wg.Done()
	}
	fn3 := func(ctx *Context) {
		wg.Done()
	}
	fn4 := func(ctx *Context) {
		wg.Done()
	}
	fn5 := func(ctx *Context) {
		t.Fatal("invalid handler called")
	}

	g1 := newGroup(r, "/api", fn1, fn2, fn3)
	g2 := g1.Group("")
	g3 := g1.Group("/users")
	g3.GET("/", fn5)
	g2.GET("/logout", fn4)
	h, _, ok := r.Match("GET", "/api/logout")
	if !ok {
		t.Fatal("expected match, none found")
	}

	h(newContext(nil, nil, nil))
	wg.Wait()
}

func TestRouterMatch(t *testing.T) {
	var err error
	cc := make(chan string, 4)
	r := newRouter()
	mediumH := func(ctx *Context) {
		cc <- "medium"
	}

	smallH := func(ctx *Context) {
		cc <- "small"
	}

	if err = r.GET(mediumRouteNoParam2, mediumH); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(smallRouteNoParam2, smallH); err != nil {
		t.Fatal(err)
	}

	handler, _, ok := r.Match("GET", "/products")
	if !ok {
		t.Fatal("invalid match, expected match and found none")
	}

	handler(nil)

	if val := <-cc; val != "small" {
		t.Fatalf("invalid handler value, expected %v and received %v", "small", val)
	}
}

func BenchmarkRouter_small(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) {})
	r.GET(mediumRoute, func(ctx *Context) {})
	r.GET(largeRoute, func(ctx *Context) {})

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", smallRouteNoParam)
	}

	b.ReportAllocs()
}

func BenchmarkRouter_medium(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) {})
	r.GET(mediumRoute, func(ctx *Context) {})
	r.GET(largeRoute, func(ctx *Context) {})

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", mediumRouteNoParam)
	}

	b.ReportAllocs()
}

func BenchmarkRouter_large(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) {})
	r.GET(mediumRoute, func(ctx *Context) {})
	r.GET(largeRoute, func(ctx *Context) {})

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", largeRouteNoParam)
	}

	b.ReportAllocs()
}
