# httpserve

HTTPServe is a simple and lightweight HTTP framework. It's intended to make HTTP controller setup fast and easy.

## Features
- Parameter and non-restrictive routing
- Middleware support
- Simple TLS configuration (http2 by default)

## Usage

```go
// Function example
package main

import (
	"log"

	"github.com/Hatch1fy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		err error
	)

	srv = httpserve.New()
	defer srv.Close()

	srv.GET("/ping", func(ctx *httpserve.Context) (res httpserve.Response) {
		return httpserve.NewTextResponse(200, []byte("pong"))
	})

	srv.Set404(func(ctx *httpserve.Context) (res httpserve.Response) {
		return httpserve.NewTextResponse(404, []byte("Oh shoot, this page doesn't exist"))
	})

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}

```

```go
// Method example
package main

import (
	"log"

	"github.com/Hatch1fy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		svc Service
		err error
	)

	srv = httpserve.New()
	defer srv.Close()

	srv.GET("/ping", svc.Ping)
	srv.Set404(svc.NotFound)

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}

// Service manages a web service
type Service struct{}

// Ping is the ping endpoint handler
func (s *Service) Ping(ctx *httpserve.Context) (res httpserve.Response) {
	return httpserve.NewTextResponse(200, []byte("pong"))
}

// NotFound is the 404 handler
func (s *Service) NotFound(ctx *httpserve.Context) (res httpserve.Response) {
	return httpserve.NewTextResponse(404, []byte("Oh shoot, this page doesn't exist"))
}

```