package apex

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

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
