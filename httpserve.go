package httpserve

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/hatchify/errors"
)

const (
	// ErrNotInitialized is returned when an action is performed on an uninitialized instance of Serve
	ErrNotInitialized = errors.Error("cannot perform action on uninitialized Serve")
	// ErrInvalidWildcardRoute is returned when an invalid wildcard route is encountered
	ErrInvalidWildcardRoute = errors.Error("wildcard routes cannot have any additional characters following the asterisk")
	// ErrMissingLeadSlash is returned when a route does not begin with "/"
	ErrMissingLeadSlash = errors.Error("invalid route, needs to start with a forward slash")
	// ErrInvalidParamLocation is returned when a parameter follows a character other than "/"
	ErrInvalidParamLocation = errors.Error("parameters can only directly follow a forward slash")
	// ErrInvalidWildcardLocation is returned when a wildcard follows a character other than "/"
	ErrInvalidWildcardLocation = errors.Error("wildcards can only directly follow a forward slash")
)

var defaultConfig = Config{
	ReadTimeout:    5 * time.Minute,
	WriteTimeout:   5 * time.Minute,
	MaxHeaderBytes: 16384,
}

// New will return a new instance of Serve
func New() *Serve {
	var s Serve
	s.g.r = newRouter()
	return &s
}

// Serve will serve HTTP requests
type Serve struct {
	s *http.Server
	g group
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

// OPTIONS will set a OPTIONS endpoint
func (s *Serve) OPTIONS(route string, hs ...Handler) {
	s.g.OPTIONS(route, hs...)
}

// Group will return a new group for a given route and handlers
func (s *Serve) Group(route string, hs ...Handler) Group {
	return s.g.Group(route, hs...)
}

// Listen will listen on a given port
func (s *Serve) Listen(port uint16) (err error) {
	return s.ListenWithConfig(port, defaultConfig)
}

// ListenWithConfig will listen on a given port using the specified configuration
func (s *Serve) ListenWithConfig(port uint16, c Config) (err error) {
	s.s = newHTTPServer(s.g.r, port, c)
	var l net.Listener
	if l, err = net.Listen("tcp", s.s.Addr); err != nil {
		return
	}

	return s.s.Serve(l)
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

	if _, err = os.Stat(certificateDir); err != nil {
		return
	}

	if tc, err = newTLSCerts(certificateDir); err != nil {
		return
	}

	if cfg.Certificates, err = tc.Certificates(); err != nil {
		return
	}

	cfg.PreferServerCipherSuites = true
	cfg.MinVersion = tls.VersionTLS12
	cfg.RootCAs = x509.NewCertPool()
	cfg.BuildNameToCertificate()
	s.s = newHTTPServer(s.g.r, port, c)
	s.s.TLSConfig = &cfg

	var l net.Listener
	if l, err = tls.Listen("tcp", s.s.Addr, &cfg); err != nil {
		return
	}

	return s.s.Serve(l)
}

// Set404 will set the 404 handler
func (s *Serve) Set404(h Handler) {
	s.g.r.SetNotFound(h)
}

// SetPanic will set the panic handler
func (s *Serve) SetPanic(h PanicHandler) {
	s.g.r.SetPanic(h)
}

// Close will close an instance of Serve
func (s *Serve) Close() (err error) {
	if s.s == nil {
		return ErrNotInitialized
	}

	return s.s.Close()
}
