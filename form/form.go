package form

import (
	"io"
	"strings"

	"github.com/gdbu/reflectio"
)

var cache = reflectio.NewCache()

// Unmarshal will parse a form and bind the values to the provided value
func Unmarshal(query string, value interface{}) (err error) {
	return BindReader(strings.NewReader(query), value)
}

// BindReader will parse a query and bind the values to the provided value
func BindReader(r io.Reader, value interface{}) (err error) {
	return NewDecoder(r).Decode(value)
}
