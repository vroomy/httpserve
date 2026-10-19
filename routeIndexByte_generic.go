//go:build (!arm64 && !amd64) || purego

package apex

import "strings"

const useRouteIndexByteSIMD = false

func routeIndexByte(value string, target byte) int {
	return strings.IndexByte(value, target)
}
