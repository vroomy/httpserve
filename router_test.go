package httpserve

import (
	"testing"

	"github.com/vroomy/common"
)

const (
	noParamRoute = "/hello/world"
)

var (
	handlerSink common.Handler
	paramsSink  Params

	boolSink bool
)

func TestRouter(t *testing.T) {
	var err error
	cc := make(chan string, 4)
	r := newRouter()
	if err = r.GET(rootRoute, func(ctx common.Context) {
		cc <- "root"
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(smallRoute, func(ctx common.Context) {
		cc <- "small"
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(mediumRoute, func(ctx common.Context) {
		cc <- "medium"
	}); err != nil {
		t.Fatal(err)
	}

	if err = r.GET(largeRoute, func(ctx common.Context) {
		cc <- "large"
	}); err != nil {
		t.Fatal(err)
	}

	fn, params, ok := r.Match("GET", rootRoute)
	if !ok {
		t.Fatal("expected match and none was found")
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

func TestRouterMatch(t *testing.T) {
	var err error
	cc := make(chan string, 4)
	r := newRouter()
	mediumH := func(ctx common.Context) {
		cc <- "medium"
	}

	smallH := func(ctx common.Context) {
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
	r.GET(smallRoute, func(ctx common.Context) {})
	r.GET(mediumRoute, func(ctx common.Context) {})
	r.GET(largeRoute, func(ctx common.Context) {})

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", smallRouteNoParam)
	}

	b.ReportAllocs()
}

func BenchmarkRouter_medium(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx common.Context) {})
	r.GET(mediumRoute, func(ctx common.Context) {})
	r.GET(largeRoute, func(ctx common.Context) {})

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", mediumRouteNoParam)
	}

	b.ReportAllocs()
}

func BenchmarkRouter_large(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx common.Context) {})
	r.GET(mediumRoute, func(ctx common.Context) {})
	r.GET(largeRoute, func(ctx common.Context) {})

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("GET", largeRouteNoParam)
	}

	b.ReportAllocs()
}
