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

func TestGetParts(t *testing.T) {
	fmt.Printf("small: %v\n", getParts(smallRoute))
	fmt.Printf("medium: %v\n", getParts(mediumRoute))
	fmt.Printf("large: %v\n", getParts(largeRoute))
}

func BenchmarkGetPartsSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = getParts(smallRoute)
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = getParts(mediumRoute)
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = getParts(largeRoute)
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
