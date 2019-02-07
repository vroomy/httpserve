package httpserve

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
)

const (
	invalidTLSFmt = "invalid tls certification pair, neither key nor cert can be empty (%s): %#v"
)

func newTLSCerts(dir string) (tc tlsCerts, err error) {
	tc = make(tlsCerts)
	err = filepath.Walk(dir, tc.walk)
	return
}

type tlsCerts map[string]*tlsCert

func (t tlsCerts) get(name string) (tc *tlsCert) {
	var ok bool
	if tc, ok = t[name]; !ok {
		tc = &tlsCert{}
		t[name] = tc
	}

	return
}
func (t tlsCerts) walk(path string, info os.FileInfo, ierr error) (err error) {
	if info.IsDir() {
		return
	}

	end := len(path) - 4
	name := path[:end]
	ext := path[end:]

	switch ext {
	case ".crt":
		t.get(name).cert = path
	case ".key":
		t.get(name).key = path
	}

	return
}

func (t tlsCerts) Certificates() (certs []tls.Certificate, err error) {
	certs = make([]tls.Certificate, 0, len(t))
	for name, tc := range t {
		if !tc.isValid() {
			err = fmt.Errorf(invalidTLSFmt, name, tc)
			return
		}

		var cert tls.Certificate
		if cert, err = tls.LoadX509KeyPair(tc.cert, tc.key); err != nil {
			return
		}

		certs = append(certs, cert)
	}

	return
}

type tlsCert struct {
	key  string
	cert string
}

func (t *tlsCert) isValid() (ok bool) {
	if len(t.key) == 0 {
		return
	}

	if len(t.cert) == 0 {
		return
	}

	return true
}
