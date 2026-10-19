package apex

import (
	"fmt"
	"strings"
	"testing"
)

func TestRouteIndexByte(t *testing.T) {
	// Vary both alignment and size around every vector and short-load boundary.
	for offset := 0; offset < 32; offset++ {
		for length := 0; length < 32; length++ {
			var data = make([]byte, offset+length+32)
			for index := range data {
				data[index] = 0x81
			}

			for _, target := range []byte{0, '/', 0x80, 0xff} {
				for position := -1; position < length; position++ {
					if position >= 0 {
						data[offset+position] = target
					}

					var value = string(data)[offset : offset+length]
					if got := routeIndexByte(value, target); got != position {
						t.Fatalf("offset=%d length=%d target=%x position=%d: got %d", offset, length, target, position, got)
					}

					if position >= 0 {
						data[offset+position] = 0x81
					}
				}
			}
		}
	}
}

func TestRouteIndexByteRepeatedAndOutsideMatches(t *testing.T) {
	for target := 0; target < 256; target++ {
		for length := 0; length < 32; length++ {
			var (
				data = make([]byte, length+2)
				want = -1
			)
			for index := range data {
				data[index] = byte(target)
			}

			var value = string(data)[1 : length+1]
			if length > 0 {
				want = 0
			}

			if got := routeIndexByte(value, byte(target)); got != want {
				t.Fatalf("target=%x length=%d: repeated match = %d, want %d", target, length, got, want)
			}

			if length >= 2 {
				data[1] = byte(target) ^ 0xff
				value = string(data)[1 : length+1]
				if got := routeIndexByte(value, byte(target)); got != 1 {
					t.Fatalf("target=%x length=%d: repeated match after first byte = %d, want 1", target, length, got)
				}
			}

			for index := 1; index <= length; index++ {
				data[index] = byte(target) ^ 0xff
			}

			value = string(data)[1 : length+1]
			if got := routeIndexByte(value, byte(target)); got != -1 {
				t.Fatalf("target=%x length=%d: matched outside string at %d", target, length, got)
			}
		}
	}
}

func FuzzRouteIndexByte(f *testing.F) {
	for _, value := range []string{"", "/", "ab", "a/b/c", strings.Repeat("x", 30) + "/", "\x00\xff\x80/\x00"} {
		f.Add(value, byte('/'))
	}

	f.Fuzz(func(t *testing.T, value string, target byte) {
		if len(value) >= 32 {
			t.Skip()
		}

		var want = strings.IndexByte(value, target)
		if got := routeIndexByte(value, target); got != want {
			t.Fatalf("routeIndexByte(%q, %x) = %d, want %d", value, target, got, want)
		}
	})
}

func BenchmarkRouteIndexByte(b *testing.B) {
	for _, length := range []int{0, 1, 2, 4, 8, 15, 16, 31} {
		for _, offset := range []int{0, 1, 15, 31} {
			for _, match := range []string{"First", "Last", "Miss"} {
				var data = []byte(strings.Repeat("x", length+64))
				if length > 0 && match == "First" {
					data[offset] = '/'
				} else if length > 0 && match == "Last" {
					data[offset+length-1] = '/'
				}

				var value = string(data)[offset : offset+length]
				b.Run(fmt.Sprintf("Bytes=%d/Offset=%d/%s", length, offset, match), func(b *testing.B) {
					for _, implementation := range []string{"Standard", "Route"} {
						b.Run(implementation, func(b *testing.B) {
							var result int
							var want = strings.IndexByte(value, '/')
							b.ReportAllocs()
							b.ResetTimer()
							if implementation == "Standard" {
								for index := 0; index < b.N; index++ {
									result = strings.IndexByte(value, '/')
								}
							} else {
								for index := 0; index < b.N; index++ {
									result = routeIndexByte(value, '/')
								}
							}

							b.StopTimer()
							if result != want {
								b.Fatalf("index = %d, want %d", result, want)
							}
						})
					}
				})
			}
		}
	}
}
