# httpserve

HTTPServe is a simple and lightweight HTTP framework. It's intended to make HTTP controller setup fast and easy.

## Features
- Parameter and non-restrictive routing
- Middleware support
- Simple TLS configuration (http2 by default)

## Routing

Routes use a compressed prefix trie for each HTTP method. Matching tries static
branches first, then named parameters, then wildcards. If a branch cannot complete
the match, the router tries the next branch. For example, `/users/current` takes
precedence over `/users/:id` regardless of registration order. Duplicate patterns
keep their first registered handler.

The trie reserves parameter storage only after a complete match, then fills
values as traversal returns and assigns names from the selected endpoint. Failed
branches neither allocate capture storage nor retain captured strings.

`Router.Match` returns caller-owned parameters with space for exactly the winning
route's captures. Static matches, parameter-free wildcard matches, and misses
need no parameter allocation. `Router.MatchInto(method, path, params)` accepts
reusable storage and returns the handler, updated parameters, and match status.
Pass its returned slice into the next call; the previous values are cleared and
overwritten. Concurrent callers must use separate backing arrays.

`ServeHTTP` reuses a pooled Context and clears parameter, hook, and storage
references before returning it to the pool. The Context and its parameter slice
are valid only during the handler. Warmed dispatch with sufficient capacity uses
zero allocations in the routing benchmarks; cold pools, response generation,
and application work can still allocate.

On ARM64, trie child selection and parameter delimiter searches use inlined
empty-string and first-byte checks, scalar loads for very small inputs, and
bounded NEON searches through 31 bytes. Longer searches keep Go's standard
library implementation. Other architectures and builds with
`-tags purego` use `strings.IndexByte`. See the
[ARM64 comparison](benchmarks/router-simd.md) for benchmark results, validation,
and reproduction commands.

## Benchmarks

For reproducible comparisons between HTTPServe and Go's `http.ServeMux`, see
[the router benchmark guide](benchmarks/README.md#after-arm64-simd-2026-10-18).
It includes fixture details, allocation measurements, and commands for comparing
the routers with `benchstat`.

### Standard library comparison

Comparison with 40 GET routes and one named parameter per route. `http.ServeMux`
shows median dispatch time; HTTPServe shows its speedup over that baseline:

| URL fixture | `http.ServeMux` | HTTPServe |
| --- | ---: | ---: |
| Common (19 bytes): `/api/users/00000000` | 122.50 ns/op | **3.24× faster** |
| Short (9 bytes) | 93.77 ns/op | **2.26× faster** |
| Medium (39 bytes) | 212.75 ns/op | **5.08× faster** |
| Long (172 bytes) | 616.20 ns/op | **14.58× faster** |

Across all four fixtures, HTTPServe used **0 B/op and 0 allocations/op**;
`http.ServeMux` used **16 B/op and 1 allocation/op**.

Measured on 2026-10-18 with Go 1.27.0 (`darwin/arm64`) on Apple M3, using one CPU
and ten one-second samples per case. All timing differences were statistically
significant in `benchstat` (`p < 0.001`). These are warmed successful-route
benchmarks that include native parameter lookup and a minimal handler, excluding
request construction, network I/O, and application work. They do not measure
end-to-end HTTP throughput or compare all routing semantics.

See the [benchmark source](router_stdlib_benchmark_test.go),
[raw samples](benchmarks/router-stdlib-simd.txt), and
[statistical comparison](benchmarks/router-stdlib-simd-comparison.txt).
The [benchmark guide](benchmarks/README.md#after-arm64-simd-2026-10-18)
includes the reproduction command and fixture details.

## Usage

```go
// Function example
package main

import (
	"log"

	"github.com/vroomy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		err error
	)

	srv = ctx.New()
	defer srv.Close()

	srv.GET("/ping", func(ctx *Context) (res vroomy.Response) {
		return ctx.NewTextResponse(200, []byte("pong"))
	})

	srv.Set404(func(ctx *Context) (res vroomy.Response) {
		return ctx.NewTextResponse(404, []byte("Oh shoot, this page doesn't exist"))
	})

	if err = srv.Listen(8080); err != nil {
		out.Errorf("error during Init: %v", err)
        return
	}
}

```

```go
// Method example
package main

import (
	"log"

	"github.com/vroomy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		svc Service
		err error
	)

	srv = ctx.New()
	defer srv.Close()

	srv.GET("/ping", svc.Ping)
	srv.Set404(svc.NotFound)

	if err = srv.Listen(8080); err != nil {
		out.Errorf("error during Init: %v", err)
        return
	}
}

// Service manages a web service
type Service struct{}

// Ping is the ping endpoint handler
func (s *Service) Ping(ctx *Context) (res vroomy.Response) {
	return ctx.NewTextResponse(200, []byte("pong"))
}

// NotFound is the 404 handler
func (s *Service) NotFound(ctx *Context) (res vroomy.Response) {
	return ctx.NewTextResponse(404, []byte("Oh shoot, this page doesn't exist"))
}

```
