package httpserve

import (
	"fmt"
	"strings"
	"testing"
)

const (
	smallRoute  = "/hello/world/:name"
	mediumRoute = "/hello/world/:name/:age"
	largeRoute  = "/hello/world/:name/:age/:occupation"

	smallRouteNoParam  = "/hello/world/name"
	mediumRouteNoParam = "/hello/world/name/age"
	largeRouteNoParam  = "/hello/world/name/age/occupation"
)

var (
	getPartsSink []string
)

func TestGetParts_small(t *testing.T) {
	ps, err := getParts(smallRoute)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_medium(t *testing.T) {
	ps, err := getParts(mediumRoute)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_large(t *testing.T) {
	ps, err := getParts(largeRoute)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Parts: %v\n", ps)
}

func TestGetParts_param(t *testing.T) {
	ps, err := getParts("/api/releases/hatch/:platform/:environment/latest")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Parts: %v\n", ps)
}

func BenchmarkGetPartsSmall(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		if getPartsSink, err = getParts(smallRoute); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsMedium(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		if getPartsSink, err = getParts(mediumRoute); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsLarge(b *testing.B) {
	var err error
	for i := 0; i < b.N; i++ {
		if getPartsSink, err = getParts(largeRoute); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = strings.Split(smallRoute, "/")
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = strings.Split(mediumRoute, "/")
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = strings.Split(largeRoute, "/")
	}

	b.ReportAllocs()
}
