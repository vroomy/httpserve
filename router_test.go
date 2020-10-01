package httpserve

import (
	"testing"
)

const (
	noParamRoute = "/hello/world"
)

var (
	handlerSink Handler
	paramsSink  Params

	boolSink bool
)

func TestRouter(t *testing.T) {
	var err error
	cc := make(chan string, 3)
	r := newRouter()
	if err = r.GET(smallRoute, func(ctx *Context) Response {
		cc <- "small"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(mediumRoute, func(ctx *Context) Response {
		cc <- "medium"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(largeRoute, func(ctx *Context) Response {
		cc <- "large"
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	fn, params, ok := r.Match("GET", smallRouteNoParam)
	if !ok {
		t.Fatal("expected match and none was found")
	}

	fn(nil)

	if val := <-cc; val != "small" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", val, "small")
	}

	fn, params, ok = r.Match("GET", mediumRouteNoParam)
	if !ok {
		t.Fatal("expected match and none was found")
	}

	fn(nil)

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

func BenchmarkRouter_small(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) Response { return nil })
	r.GET(mediumRoute, func(ctx *Context) Response { return nil })
	r.GET(largeRoute, func(ctx *Context) Response { return nil })

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", smallRouteNoParam)
	}

	b.ReportAllocs()
}

func BenchmarkRouter_medium(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) Response { return nil })
	r.GET(mediumRoute, func(ctx *Context) Response { return nil })
	r.GET(largeRoute, func(ctx *Context) Response { return nil })

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", mediumRouteNoParam)
	}

	b.ReportAllocs()
}

func BenchmarkRouter_large(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) Response { return nil })
	r.GET(mediumRoute, func(ctx *Context) Response { return nil })
	r.GET(largeRoute, func(ctx *Context) Response { return nil })

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", largeRouteNoParam)
	}

	b.ReportAllocs()
}
