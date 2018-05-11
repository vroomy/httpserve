package httpserve

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	// TODO: See if this is still needed
	"github.com/bradfitz/http2"
	"github.com/julienschmidt/httprouter"
)

const (
	invalidTLSFmt = "invalid tls certification pair, neither key nor cert can be empty (%s): %#v"
)

var defaultConfig = Config{
	ReadTimeout:    5 * time.Minute,
	WriteTimeout:   5 * time.Minute,
	MaxHeaderBytes: 16384,
}

// New will return a new instance of Serve
func New() *Serve {
	var s Serve
	s.g.r = httprouter.New()
	return &s
}

// Serve will serve HTTP requests
type Serve struct {
	g Group
}

// GET will set a GET endpoint
func (s *Serve) GET(route string, hs ...Handler) {
	s.g.GET(route, hs...)
}

// PUT will set a PUT endpoint
func (s *Serve) PUT(route string, hs ...Handler) {
	s.g.PUT(route, hs...)
}

// POST will set a POST endpoint
func (s *Serve) POST(route string, hs ...Handler) {
	s.g.POST(route, hs...)
}

// DELETE will set a DELETE endpoint
func (s *Serve) DELETE(route string, hs ...Handler) {
	s.g.DELETE(route, hs...)
}

// Group will return a new group for a given route and handlers
func (s *Serve) Group(route string, hs ...Handler) *Group {
	return s.g.Group(route, hs...)
}

// Listen will listen on a given port
func (s *Serve) Listen(port uint16) (err error) {
	return s.ListenWithConfig(port, defaultConfig)
}

// ListenWithConfig will listen on a given port using the specified configuration
func (s *Serve) ListenWithConfig(port uint16, c Config) (err error) {
	srv := newHTTPServer(s.g.r, port, c)
	var l net.Listener
	if l, err = net.Listen("tcp", srv.Addr); err != nil {
		return
	}

	return srv.Serve(l)
}

// ListenTLS will listen using the TLS procol on a given port
func (s *Serve) ListenTLS(port uint16, certificateDir string) (err error) {
	return s.ListenTLSWithConfig(port, certificateDir, defaultConfig)
}

// ListenTLSWithConfig will listen using the TLS procol on a given port using the specified configuration
func (s *Serve) ListenTLSWithConfig(port uint16, certificateDir string, c Config) (err error) {
	var (
		tc  tlsCerts
		cfg tls.Config
	)

	if tc, err = newTLSCerts(certificateDir); err != nil {
		return
	}

	if cfg.Certificates, err = tc.Certificates(); err != nil {
		return
	}

	cfg.RootCAs = x509.NewCertPool()
	cfg.BuildNameToCertificate()
	srv := newHTTPServer(s.g.r, port, c)
	srv.TLSConfig = &cfg
	http2.ConfigureServer(&srv, &http2.Server{})

	var l net.Listener
	if l, err = tls.Listen("tcp", srv.Addr, &cfg); err != nil {
		return
	}

	return srv.Serve(l)
}
