//go:build !arm64 || purego

package httpserve

import "strings"

const useRouteIndexByteNEON = false

func routeIndexByte(value string, target byte) int {
	return strings.IndexByte(value, target)
}
