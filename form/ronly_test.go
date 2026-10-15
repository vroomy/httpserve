package form

import "io"

type ronly struct {
	r io.Reader
}

func (r *ronly) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

var _ io.Reader = &ronly{}
