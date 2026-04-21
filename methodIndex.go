package httpserve

import "net/http"

// methodIndex maps HTTP method strings to a compact array index,
// replacing the map[string]routes lookup with a direct array access.
type methodIndex uint8

const (
	methodUnknown methodIndex = iota // 0 — zero value, safe default for uninitialized
	methodGET                        // 1
	methodHEAD                       // 2
	methodPOST                       // 3
	methodPUT                        // 4
	methodDELETE                     // 5
	methodOPTIONS                    // 6
	numMethods    = 7
)

func methodToIndex(m string) methodIndex {
	switch m {
	case http.MethodGet:
		return methodGET
	case http.MethodHead:
		return methodHEAD
	case http.MethodPost:
		return methodPOST
	case http.MethodPut:
		return methodPUT
	case http.MethodDelete:
		return methodDELETE
	case http.MethodOptions:
		return methodOPTIONS
	default:
		return methodUnknown
	}
}
