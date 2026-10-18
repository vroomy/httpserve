//go:build !purego

package httpserve

const useRouteIndexByteSIMD = true

// routeIndexByte searches strings shorter than 32 bytes. It avoids the assembly
// call for empty edges and first-byte hits.
// Keep this wrapper inlinable: another Go call regresses deep trie lookups.
func routeIndexByte(value string, target byte) int {
	if len(value) == 0 {
		return -1
	}

	if value[0] == target {
		return 0
	}

	return routeIndexByteNEON(value, target)
}

// routeIndexByteNEON searches 1..31 bytes whose first byte does not match.
// It uses scalar loads below four bytes and bounded NEON loads otherwise.
//
//go:noescape
func routeIndexByteNEON(value string, target byte) int
