package httpserve

import (
	"strings"
)

func newParam(part, url string) (param Param, n int) {
	if n = strings.IndexByte(url, '/'); n == -1 {
		n = len(url)
	}

	param.Key = part[1:]
	param.Value = url[:n]
	return
}

// Param represents a key/value pair
type Param struct {
	Key   string
	Value string
}
