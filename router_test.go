package httpserve

import (
	"net/http"
	"testing"

	"github.com/julienschmidt/httprouter"
	as "github.com/missionMeteora/apiserv/router"
)

const (
	noParamRoute = "/hello/world"
)

var (
	handlerSink Handler
	paramsSink  Params

	jsHandleSink httprouter.Handle
	jsParamsSink httprouter.Params

	asHandlerSink as.Handler
	asParamsSink  as.Params

	boolSink bool
)

func TestRouter(t *testing.T) {
	r := newRouter()
	r.GET(smallRouteNoParam, func(ctx *Context) Response { return nil })
	r.GET(smallRoute, func(ctx *Context) Response { return nil })

	_, params, ok := r.Match("/hello/world")
	if !ok {
		t.Fatal("expected match and none was found")
	}

	_, params, ok = r.Match("/hello/world/Josh")
	if !ok {
		t.Fatal("expected match and none was found")
	}

	if params["name"] != "Josh" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", "Josh", params["name"])
	}

	if !ok {
		t.Fatalf("invalid OK value")
	}
}

func BenchmarkRouter(b *testing.B) {
	r := newRouter()
	r.GET(smallRoute, func(ctx *Context) Response { return nil })

	for i := 0; i < b.N; i++ {
		handlerSink, paramsSink, boolSink = r.Match("/hello/world/Josh")
	}

	b.ReportAllocs()
}

func BenchmarkJulianSchmidt(b *testing.B) {
	r := httprouter.New()
	r.GET(smallRoute, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {})

	for i := 0; i < b.N; i++ {
		jsHandleSink, jsParamsSink, boolSink = r.Lookup("GET", "/hello/world/Josh")
	}

	b.ReportAllocs()
}

func BenchmarkAPIServe(b *testing.B) {
	r := as.New(nil)

	r.GET(smallRoute, func(w http.ResponseWriter, r *http.Request, p as.Params) {})

	for i := 0; i < b.N; i++ {
		asHandlerSink, asParamsSink = r.Match("GET", "/hello/world/Josh")
	}

	b.ReportAllocs()
}
