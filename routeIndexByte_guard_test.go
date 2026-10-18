//go:build (arm64 || amd64) && !purego && (darwin || linux)

package httpserve

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func TestRouteIndexByteGuardPages(t *testing.T) {
	var (
		pageSize = os.Getpagesize()
		pages    []byte
		err      error
	)
	if pages, err = syscall.Mmap(-1, 0, 3*pageSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE); err != nil {
		t.Fatalf("map guard pages: %v", err)
	}

	t.Cleanup(func() {
		if cleanupErr := syscall.Munmap(pages); cleanupErr != nil {
			t.Errorf("unmap guard pages: %v", cleanupErr)
		}
	})
	if err = syscall.Mprotect(pages[:pageSize], syscall.PROT_NONE); err != nil {
		t.Fatalf("protect first page: %v", err)
	}

	if err = syscall.Mprotect(pages[2*pageSize:], syscall.PROT_NONE); err != nil {
		t.Fatalf("protect last page: %v", err)
	}

	var data = pages[pageSize : 2*pageSize]
	for index := range data {
		data[index] = 0x81
	}

	for length := 0; length < 32; length++ {
		for _, start := range []int{0, pageSize - length} {
			var value = unsafe.String(unsafe.SliceData(data[start:]), length)
			for _, target := range []byte{0, '/', 0xff} {
				if got := routeIndexByte(value, target); got != -1 {
					t.Fatalf("start=%d length=%d target=%x: miss returned %d", start, length, target, got)
				}

				if length == 0 {
					continue
				}

				data[start+length-1] = target
				if got := routeIndexByte(value, target); got != length-1 {
					t.Fatalf("start=%d length=%d target=%x: last byte returned %d", start, length, target, got)
				}

				data[start+length-1] = 0x81
			}
		}
	}
}
