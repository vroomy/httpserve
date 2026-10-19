package apex

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	ReadTimeout:    30 * time.Second,
	WriteTimeout:   30 * time.Second,
	IdleTimeout:    90 * time.Second,
	MaxHeaderBytes: 4096,
}

var _ Group = &Engine{}

// New constructs an Engine with an empty router.
func New() *Engine {
	var e Engine
	e.g.r = newRouter()
	return &e
}

// Engine manages routing, middleware, and HTTP/HTTPS listeners.
type Engine struct {
	mu     sync.Mutex
	closed bool
	http   *http.Server
	https  *http.Server
	g      group
}

// GET will set a GET endpoint
func (e *Engine) GET(route string, hs ...Handler) (err error) {
	return e.g.GET(route, hs...)
}

// PUT will set a PUT endpoint
func (e *Engine) PUT(route string, hs ...Handler) (err error) {
	return e.g.PUT(route, hs...)
}

// POST will set a POST endpoint
func (e *Engine) POST(route string, hs ...Handler) (err error) {
	return e.g.POST(route, hs...)
}

// DELETE will set a DELETE endpoint
func (e *Engine) DELETE(route string, hs ...Handler) (err error) {
	return e.g.DELETE(route, hs...)
}

// OPTIONS will set a OPTIONS endpoint
func (e *Engine) OPTIONS(route string, hs ...Handler) (err error) {
	return e.g.OPTIONS(route, hs...)
}

// Handle will create a route for any method
func (e *Engine) Handle(method, route string, hs ...Handler) (err error) {
	return e.g.Handle(method, route, hs...)
}

// Group will return a new group for a given route and handlers
func (e *Engine) Group(route string, hs ...Handler) Group {
	return e.g.Group(route, hs...)
}

// Listen will listen on a given port
func (e *Engine) Listen(port uint16) (err error) {
	return e.ListenWithConfig(port, defaultConfig)
}

// ListenWithConfig will listen on a given port using the specified configuration
func (e *Engine) ListenWithConfig(port uint16, c Config) (err error) {
	server := newHTTPServer(e.g.r, port, c)
	if _, enabled := os.LookupEnv("USE_HTTP2_RAW"); enabled {
		server.Handler = h2c.NewHandler(server.Handler, &http2.Server{})
	}

	return e.listenServer(server, false)
}

// ListenTLS will listen using the TLS procol on a given port
func (e *Engine) ListenTLS(port uint16, certificateDir string) (err error) {
	return e.ListenTLSWithConfig(port, certificateDir, defaultConfig)
}

// ListenTLSWithConfig will listen using the TLS procol on a given port using the specified configuration
func (e *Engine) ListenTLSWithConfig(port uint16, certificateDir string, c Config) (err error) {
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

	//Enable http2 as protocol option
	cfg.NextProtos = []string{"h2", "http/1.1"}

	cfg.MinVersion = tls.VersionTLS12
	cfg.RootCAs = x509.NewCertPool()
	server := newHTTPServer(e.g.r, port, c)
	server.TLSConfig = &cfg
	return e.listenServer(server, true)
}

// ListenAutoCertTLS will listen using the TLS procol on a given port using the certificate being provided by LetsEncrypt
func (e *Engine) ListenAutoCertTLS(port uint16, ac AutoCertConfig) (err error) {
	return e.ListenAutoCertTLSWithConfig(port, ac, defaultConfig)
}

// ListenAutoCertTLSWithConfig will listen using the TLS procol on a given port using configurations and the certificate being provided by LetsEncrypt
func (e *Engine) ListenAutoCertTLSWithConfig(port uint16, ac AutoCertConfig, c Config) (err error) {
	server := newHTTPServer(e.g.r, port, c)

	m := &autocert.Manager{
		Cache:      autocert.DirCache(ac.DirCache),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: ac.hostPolicy(),
	}

	cfg := m.TLSConfig()
	cfg.MinVersion = tls.VersionTLS12
	server.TLSConfig = cfg
	return e.listenServer(server, true)
}

// Set404 will set the 404 handler
func (e *Engine) Set404(h Handler) {
	e.g.r.SetNotFound(h)
}

// SetPanic will set the panic handler
func (e *Engine) SetPanic(h PanicHandler) {
	e.g.r.SetPanic(h)
}

// SetPanic will set the panic handler
func (e *Engine) SetOnError(fn func(error)) {
	e.g.r.SetOnError(fn)
}

// Close closes both servers and prevents listeners from starting afterward.
func (e *Engine) Close() error {
	e.mu.Lock()
	e.closed = true
	httpServer, httpsServer := e.http, e.https
	e.mu.Unlock()

	var errs []error
	if httpServer != nil {
		errs = append(errs, httpServer.Close())
	}

	if httpsServer != nil {
		errs = append(errs, httpsServer.Close())
	}

	return errors.Join(errs...)
}

func (e *Engine) listenServer(server *http.Server, secure bool) error {
	var (
		listener net.Listener
		err      error
	)
	if listener, err = e.openListener(server, secure); err != nil {
		return err
	}

	if secure {
		return server.ServeTLS(listener, "", "")
	}

	return server.Serve(listener)
}

// openListener publishes server ownership and binds under the startup/Close lock.
// Serving begins after the lock is released; http.Server handles Close racing with Serve.
func (e *Engine) openListener(server *http.Server, secure bool) (net.Listener, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil, http.ErrServerClosed
	}

	destination := &e.http
	if secure {
		destination = &e.https
	}

	if *destination != nil {
		return nil, fmt.Errorf("listener already started for %s", server.Addr)
	}

	*destination = server
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", server.Addr); err != nil {
		return nil, fmt.Errorf("listen on %s: %w", server.Addr, err)
	}

	return listener, nil
}
