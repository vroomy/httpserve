package httpserve

import (
	"fmt"
	"strings"
	"testing"
)

const (
	getPartsSmall  = "/hello/world"
	getPartsMedium = "/hello/world/:name"
	getPartsLarge  = "/hello/world/:name/:age"
)

var (
	getPartsSink []string
)

func TestGetParts(t *testing.T) {
	fmt.Printf("small: %v\n", getParts(getPartsSmall))
	fmt.Printf("medium: %v\n", getParts(getPartsMedium))
	fmt.Printf("large: %v\n", getParts(getPartsLarge))
}

func BenchmarkGetPartsSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = getParts(getPartsSmall)
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = getParts(getPartsMedium)
	}

	b.ReportAllocs()
}

func BenchmarkGetPartsLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = getParts(getPartsLarge)
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = strings.Split(getPartsSmall, "/")
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = strings.Split(getPartsMedium, "/")
	}

	b.ReportAllocs()
}

func BenchmarkStringsSpitLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		getPartsSink = strings.Split(getPartsLarge, "/")
	}

	b.ReportAllocs()
}
