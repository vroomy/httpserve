package apex

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEngineRouteMethods(t *testing.T) {
	var engine = New()
	var methods = []struct {
		name     string
		register func(string, ...Handler) error
	}{
		{"GET", engine.GET},
		{"PUT", engine.PUT},
		{"POST", engine.POST},
		{"DELETE", engine.DELETE},
		{"OPTIONS", engine.OPTIONS},
		{"HEAD", func(path string, handlers ...Handler) error { return engine.Handle("HEAD", path, handlers...) }},
	}
	for _, method := range methods {
		t.Run(method.name, func(t *testing.T) {
			var err = method.register("/items/:id", func(c *Context) { c.WriteString(202, "text/plain", c.Param("id")) })
			if err != nil {
				t.Fatal(err)
			}

			var recorder = httptest.NewRecorder()
			engine.g.r.ServeHTTP(recorder, httptest.NewRequest(method.name, "/items/42", nil))
			if recorder.Code != 202 || recorder.Body.String() != "42" {
				t.Fatalf("response = %d/%q", recorder.Code, recorder.Body.String())
			}

			if err = method.register("invalid", func(*Context) {}); err == nil {
				t.Fatal("registration accepted a path without a leading slash")
			}
		})
	}
}

func TestEngineCallbacks(t *testing.T) {
	var engine = New()
	engine.Set404(func(c *Context) { c.WriteString(404, "text/plain", "custom missing") })
	var recorder = httptest.NewRecorder()
	engine.g.r.ServeHTTP(recorder, httptest.NewRequest("GET", "/missing", nil))
	if recorder.Code != 404 || recorder.Body.String() != "custom missing" {
		t.Fatalf("not found = %d/%q", recorder.Code, recorder.Body.String())
	}

	var recovered any
	engine.SetPanic(func(value any) { recovered = value })
	if err := engine.GET("/panic", func(*Context) { panic("test panic") }); err != nil {
		t.Fatal(err)
	}

	recorder = httptest.NewRecorder()
	engine.g.r.ServeHTTP(recorder, httptest.NewRequest("GET", "/panic", nil))
	if recovered != "test panic" || recorder.Code != 500 {
		t.Fatalf("panic callback = %v, status = %d", recovered, recorder.Code)
	}

	var reported error
	engine.SetOnError(func(err error) { reported = err })
	if err := engine.GET("/error", func(c *Context) {
		c.WriteNoContent()
		c.WriteNoContent()
	}); err != nil {
		t.Fatal(err)
	}

	engine.g.r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/error", nil))
	if !errors.Is(reported, ErrContextIsClosed) {
		t.Fatalf("error callback = %v", reported)
	}
}

// waitForServer reads published ownership under the same lock as startup.
// openListener publishes and binds while holding this lock.
func waitForServer(t *testing.T, engine *Engine, secure bool, result <-chan error) *http.Server {
	t.Helper()
	var timeout = time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	var ticker = time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		engine.mu.Lock()
		var server = engine.http
		if secure {
			server = engine.https
		}

		engine.mu.Unlock()
		if server != nil {
			return server
		}

		select {
		case err := <-result:
			t.Fatalf("listener exited before startup: %v", err)
		case <-timeout.C:
			t.Fatal("listener did not start")
		case <-ticker.C:
		}
	}
}

func closeTestEngine(t *testing.T, engine *Engine, result <-chan error) {
	t.Helper()
	if err := engine.Close(); err != nil {
		t.Error(err)
	}

	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("listener returned %v, want ErrServerClosed", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("listener did not exit after Close")
	}
}

func TestEngineHTTPConfiguration(t *testing.T) {
	t.Setenv("USE_HTTP2_RAW", "1")
	var engine = New()
	var config = Config{ReadTimeout: time.Second, WriteTimeout: 2 * time.Second, IdleTimeout: 3 * time.Second, MaxHeaderBytes: 8192}
	var result = make(chan error, 1)
	go func() { result <- engine.ListenWithConfig(0, config) }()
	t.Cleanup(func() { closeTestEngine(t, engine, result) })
	var server = waitForServer(t, engine, false, result)
	if server.Protocols == nil || !server.Protocols.HTTP1() || !server.Protocols.UnencryptedHTTP2() {
		t.Fatal("raw HTTP/2 configuration missing HTTP/1 or unencrypted HTTP/2")
	}

	if server.Handler != engine.g.r || server.Addr != ":0" || server.ReadTimeout != config.ReadTimeout || server.WriteTimeout != config.WriteTimeout || server.IdleTimeout != config.IdleTimeout || server.MaxHeaderBytes != config.MaxHeaderBytes || !server.DisableGeneralOptionsHandler {
		t.Fatalf("server configuration = %#v", server)
	}
}

func TestEngineTLSConfiguration(t *testing.T) {
	var directory = t.TempDir()
	writeTestCertificate(t, directory, "localhost")
	var engine = New()
	var result = make(chan error, 1)
	go func() { result <- engine.ListenTLS(0, directory) }()
	t.Cleanup(func() { closeTestEngine(t, engine, result) })
	var server = waitForServer(t, engine, true, result)
	if server.TLSConfig.MinVersion != tls.VersionTLS12 || len(server.TLSConfig.Certificates) != 1 || server.TLSConfig.RootCAs == nil {
		t.Fatalf("TLS configuration = %#v", server.TLSConfig)
	}
}

func TestEngineTLSSetupErrors(t *testing.T) {
	var directory = t.TempDir()
	var engine = New()
	if err := engine.ListenTLSWithConfig(0, filepath.Join(directory, "missing"), defaultConfig); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing directory error = %v", err)
	}

	if err := os.WriteFile(filepath.Join(directory, "broken.crt"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := engine.ListenTLSWithConfig(0, directory, defaultConfig); err == nil {
		t.Fatal("incomplete certificate accepted")
	}

	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestEngineTLSDirectoryWalkFailure(t *testing.T) {
	var directory = t.TempDir()
	var restricted = filepath.Join(directory, "restricted")
	if err := os.Mkdir(restricted, 0700); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.Chmod(restricted, 0700); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chmod(restricted, 0); err != nil {
		t.Fatal(err)
	}

	if _, err := os.ReadDir(restricted); !errors.Is(err, os.ErrPermission) {
		t.Skip("this environment does not enforce directory read permissions")
	}

	var engine = New()
	if err := engine.ListenTLSWithConfig(0, directory, defaultConfig); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("directory walk error = %v, want permission cause", err)
	}
}

func TestEngineAutoCertConfiguration(t *testing.T) {
	var engine = New()
	var result = make(chan error, 1)
	var directory = t.TempDir()
	go func() {
		result <- engine.ListenAutoCertTLS(0, AutoCertConfig{DirCache: directory, Hosts: []string{"localhost"}})
	}()
	t.Cleanup(func() { closeTestEngine(t, engine, result) })
	var server = waitForServer(t, engine, true, result)
	if server.TLSConfig.MinVersion != tls.VersionTLS12 || server.TLSConfig.GetCertificate == nil {
		t.Fatalf("automatic TLS configuration = %#v", server.TLSConfig)
	}
}

func TestEngineListenerOwnership(t *testing.T) {
	var engine = New()
	var server = &http.Server{Addr: "127.0.0.1:0", Handler: engine.g.r}
	var listener, err = engine.openListener(server, false)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		// The listener has not been transferred to Serve in this test.
		if closeErr := listener.Close(); closeErr != nil {
			t.Error(closeErr)
		}

		if closeErr := engine.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	if duplicate, duplicateErr := engine.openListener(&http.Server{Addr: "127.0.0.1:0"}, false); duplicateErr == nil || duplicate != nil {
		t.Fatalf("duplicate startup = %v/%v", duplicate, duplicateErr)
	}

	var other = New()
	if occupied, occupiedErr := other.openListener(&http.Server{Addr: listener.Addr().String()}, false); occupiedErr == nil || occupied != nil {
		t.Fatalf("occupied address = %v/%v", occupied, occupiedErr)
	}

	if err = other.Close(); err != nil {
		t.Fatal(err)
	}

	if err = other.ListenWithConfig(0, defaultConfig); !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("startup after close = %v", err)
	}
}

func TestEngineLoopbackHTTP(t *testing.T) {
	var engine = New()
	if err := engine.GET("/ping", func(c *Context) { c.WriteString(200, "text/plain", "pong") }); err != nil {
		t.Fatal(err)
	}

	var listener net.Listener
	var err error
	var server = &http.Server{Addr: "127.0.0.1:0", Handler: engine.g.r}
	if listener, err = engine.openListener(server, false); err != nil {
		t.Fatal(err)
	}

	var result = make(chan error, 1)
	go func() { result <- server.Serve(listener) }()
	t.Cleanup(func() { closeTestEngine(t, engine, result) })
	var client = &http.Client{Timeout: 5 * time.Second}
	var response *http.Response
	if response, err = client.Get("http://" + listener.Addr().String() + "/ping"); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}

		client.CloseIdleConnections()
	}()
	var body []byte
	if body, err = io.ReadAll(response.Body); err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != 200 || string(body) != "pong" {
		t.Fatalf("loopback response = %d/%q", response.StatusCode, body)
	}
}
