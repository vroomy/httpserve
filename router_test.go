package httpserve

import (
	"net/http"
	"testing"

	"github.com/julienschmidt/httprouter"
	as "github.com/missionMeteora/apiserv/router"
)

const (
	basicRoute = "/hello/world/:name"
)

var (
	routeSink  *route
	paramsSink Params

	jsHandleSink httprouter.Handle
	jsParamsSink httprouter.Params

	asHandlerSink as.Handler
	asParamsSink  as.Params

	boolSink bool
)

func TestRouter(t *testing.T) {
	r := newRouter()
	r.GET(basicRoute, func(ctx *Context) Response { return nil })

	rt, params := r.check("/hello/world/Josh")
	if rt == nil {
		t.Fatal("expected match and none was found")
	}

	if params["name"] != "Josh" {
		t.Fatalf("invalid value, expected \"%s\" and received \"%s\"", "Josh", params["name"])
	}
}

func BenchmarkRouter(b *testing.B) {
	r := newRouter()
	r.GET(basicRoute, func(ctx *Context) Response { return nil })

	for i := 0; i < b.N; i++ {
		routeSink, paramsSink = r.check("/hello/world/Josh")
	}

	b.ReportAllocs()
}

func BenchmarkJulianSchmidt(b *testing.B) {
	r := httprouter.New()
	r.GET(basicRoute, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {})

	for i := 0; i < b.N; i++ {
		jsHandleSink, jsParamsSink, boolSink = r.Lookup("GET", "/hello/world/Josh")
	}

	b.ReportAllocs()
}

func BenchmarkAPIServe(b *testing.B) {
	r := as.New(nil)

	r.GET(basicRoute, func(w http.ResponseWriter, r *http.Request, p as.Params) {})

	for i := 0; i < b.N; i++ {
		asHandlerSink, asParamsSink = r.Match("GET", "/hello/world/Josh")
	}

	b.ReportAllocs()
}
