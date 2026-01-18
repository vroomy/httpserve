package httpserve

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

var (
	// ErrInvalidWildcardRoute is returned when an invalid wildcard route is encountered
	ErrInvalidWildcardRoute = errors.New("wildcard routes cannot have any additional characters following the asterisk")
	// ErrMissingLeadSlash is returned when a route does not begin with "/"
	ErrMissingLeadSlash = errors.New("invalid route, needs to start with a forward slash")
	// ErrInvalidParamLocation is returned when a parameter follows a character other than "/"
	ErrInvalidParamLocation = errors.New("parameters can only directly follow a forward slash")
	// ErrInvalidWildcardLocation is returned when a wildcard follows a character other than "/"
	ErrInvalidWildcardLocation = errors.New("wildcards can only directly follow a forward slash")
	// ErrContextIsClosed is returned when write actions are attempted on a closed context
	ErrContextIsClosed = errors.New("cannot perform write actions on a closed context")
)

var defaultConfig = Config{
	ReadTimeout:    5 * time.Minute,
	WriteTimeout:   5 * time.Minute,
	MaxHeaderBytes: 16384,
}

var _ Group = &Serve{}

// New will return a new instance of Serve
func New() *Serve {
	var s Serve
	s.g.r = newRouter()
	return &s
}

// Serve will serve HTTP requests
type Serve struct {
	http  *http.Server
	https *http.Server
	g     group
}

// GET will set a GET endpoint
func (s *Serve) GET(route string, hs ...Handler) (err error) {
	return s.g.GET(route, hs...)
}

// PUT will set a PUT endpoint
func (s *Serve) PUT(route string, hs ...Handler) (err error) {
	return s.g.PUT(route, hs...)
}

// POST will set a POST endpoint
func (s *Serve) POST(route string, hs ...Handler) (err error) {
	return s.g.POST(route, hs...)
}

// DELETE will set a DELETE endpoint
func (s *Serve) DELETE(route string, hs ...Handler) (err error) {
	return s.g.DELETE(route, hs...)
}

// OPTIONS will set a OPTIONS endpoint
func (s *Serve) OPTIONS(route string, hs ...Handler) (err error) {
	return s.g.OPTIONS(route, hs...)
}

// Handle will create a route for any method
func (s *Serve) Handle(method, route string, hs ...Handler) (err error) {
	return s.g.Handle(method, route, hs...)
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
	s.http = newHTTPServer(s.g.r, port, c)
	var l net.Listener
	if l, err = net.Listen("tcp", s.http.Addr); err != nil {
		return
	}

	return s.http.Serve(l)
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

	cfg.MinVersion = tls.VersionTLS12
	cfg.RootCAs = x509.NewCertPool()
	s.https = newHTTPServer(s.g.r, port, c)
	s.https.TLSConfig = &cfg

	var l net.Listener
	if l, err = tls.Listen("tcp", s.https.Addr, &cfg); err != nil {
		return
	}

	return s.https.Serve(l)
}

// ListenAutoCertTLS will listen using the TLS procol on a given port using the certificate being provided by LetsEncrypt
func (s *Serve) ListenAutoCertTLS(port uint16, ac AutoCertConfig) (err error) {
	return s.ListenAutoCertTLSWithConfig(port, ac, defaultConfig)
}

// ListenAutoCertTLSWithConfig will listen using the TLS procol on a given port using configurations and the certificate being provided by LetsEncrypt
func (s *Serve) ListenAutoCertTLSWithConfig(port uint16, ac AutoCertConfig, c Config) (err error) {
	s.https = newHTTPServer(s.g.r, port, c)

	m := &autocert.Manager{
		Cache:      autocert.DirCache(ac.DirCache),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: ac.hostPolicy(),
	}

	cfg := m.TLSConfig()
	cfg.MinVersion = tls.VersionTLS12
	s.https.TLSConfig = cfg
	return s.https.ListenAndServeTLS("", "")
}

// Set404 will set the 404 handler
func (s *Serve) Set404(h Handler) {
	s.g.r.SetNotFound(h)
}

// SetPanic will set the panic handler
func (s *Serve) SetPanic(h PanicHandler) {
	s.g.r.SetPanic(h)
}

// SetPanic will set the panic handler
func (s *Serve) SetOnError(fn func(error)) {
	s.g.r.SetOnError(fn)
}

// Close will close an instance of Serve
func (s *Serve) Close() (err error) {
	var errs []error
	if s.http != nil {
		errs = append(errs, s.http.Close())
	}

	if s.https != nil {
		errs = append(errs, s.https.Close())
	}

	return errors.Join(errs...)
}
