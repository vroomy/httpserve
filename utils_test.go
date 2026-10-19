package apex

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

func TestDefaultNotFoundAndPanic(t *testing.T) {
	var router = newRouter()
	var recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/missing", nil))
	if recorder.Code != 404 || recorder.Body.String() != "404, not found" || recorder.Header().Get("Content-Type") != "text/plain" {
		t.Fatalf("default not found = %d/%q", recorder.Code, recorder.Body.String())
	}

	var output bytes.Buffer
	var previous = log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	if err := router.GET("/panic", func(*Context) { panic("default panic") }); err != nil {
		t.Fatal(err)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/panic", nil))
	if recorder.Code != 500 || !strings.Contains(output.String(), "default panic") {
		t.Fatalf("default panic = %d/%q", recorder.Code, output.String())
	}
}

func TestCloseAndReportFailure(t *testing.T) {
	var output bytes.Buffer
	var previous = log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	var body = &trackedBody{Reader: strings.NewReader(""), err: errors.New("body close failed")}
	closeAndReport(body)
	if !body.closed || !strings.Contains(output.String(), "body close failed") {
		t.Fatalf("closed = %t, log = %q", body.closed, output.String())
	}
}

func TestGetPartsNormalization(t *testing.T) {
	var cases = []struct {
		path string
		want []string
	}{
		{path: "/", want: []string{"/"}},
		{path: "///"},
		{path: "/api//items/", want: []string{"/api/items"}},
		{path: "//api///:id//tail/*/:ignored/", want: []string{"/api", ":id", "/tail", "*", ":ignored"}},
		{path: "/:a/:b/", want: []string{":a", ":b"}},
	}
	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) {
			var parts = getParts(test.path)
			if !slices.Equal(parts, test.want) {
				t.Fatalf("getParts = %v, want %v", parts, test.want)
			}
		})
	}
}

const (
	smallRoute  = "/hello/world/:name"
	mediumRoute = "/hello/world/:name/:age"
	largeRoute  = "/hello/world/:name/:age/:occupation"

	rootRoute          = "/"
	smallRouteNoParam  = "/hello/world/name"
	mediumRouteNoParam = "/hello/world/name/age"
	largeRouteNoParam  = "/hello/world/name/age/occupation"

	mediumRouteNoParam2 = "/product/:productName"
	smallRouteNoParam2  = "/products"
)

var (
	getPartsSink []string
)

func TestGetParts_root(t *testing.T) {
	var ps = getParts(rootRoute)
	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_small(t *testing.T) {
	var ps = getParts(smallRoute)
	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_medium(t *testing.T) {
	var ps = getParts(mediumRoute)
	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_large(t *testing.T) {
	var ps = getParts(largeRoute)
	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_param(t *testing.T) {
	var ps = getParts("/api/releases/hatch/:platform/:environment/latest")
	fmt.Printf("Parts: %v\n", ps)
}

func BenchmarkGetPartsSmall(b *testing.B) {
	for b.Loop() {
		getPartsSink = getParts(smallRoute)
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsMedium(b *testing.B) {
	for b.Loop() {
		getPartsSink = getParts(mediumRoute)
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsLarge(b *testing.B) {
	for b.Loop() {
		getPartsSink = getParts(largeRoute)
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitSmall(b *testing.B) {
	for b.Loop() {
		getPartsSink = strings.Split(smallRoute, "/")
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitMedium(b *testing.B) {
	for b.Loop() {
		getPartsSink = strings.Split(mediumRoute, "/")
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitLarge(b *testing.B) {
	for b.Loop() {
		getPartsSink = strings.Split(largeRoute, "/")
	}

	b.ReportAllocs()
}
