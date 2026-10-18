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

Trie prefix comparisons accept empty prefixes directly and compare one-byte
prefixes after checking for an empty URL. Longer prefixes use `strings.HasPrefix`.
These paths apply on every architecture, including purego builds. See the
[prefix comparison report](benchmarks/trie-prefix/README.md) for validation and
measurements.

`ServeHTTP` reuses a pooled Context and clears parameter, hook, and storage
references before returning it to the pool. The Context and its parameter slice
are valid only during the handler. Warmed dispatch with sufficient capacity uses
zero allocations in the routing benchmarks; cold pools, response generation,
and application work can still allocate.

On ARM64 and AMD64, trie child selection and parameter delimiter searches use
inlined empty-string and first-byte checks, scalar loads for very small inputs, and
bounded SIMD searches through 31 bytes (NEON on ARM64, SSE2 on AMD64). Longer
searches keep Go's standard library implementation. Other architectures and
builds with `-tags purego` use `strings.IndexByte`. See the
[ARM64 comparison](benchmarks/router-simd.md) and
[AMD64 validation](benchmarks/router-simd-amd64.md) for measurements and
reproduction commands. AMD64 execution was validated under Rosetta; native
Intel/AMD performance measurements remain outstanding.

## Benchmarks

For reproducible comparisons between HTTPServe and Go's `http.ServeMux`, see
[the router benchmark guide](benchmarks/README.md#prefix-branch-versus-main-and-stdlib-2026-10-18).
It includes fixture details, allocation measurements, and commands for comparing
the routers with `benchstat`.

### Standard library comparison

Comparison with 40 GET routes and one named parameter per route. Timings are
median dispatch times; speedup is `http.ServeMux` time divided by HTTPServe time:

| URL fixture | `http.ServeMux` | HTTPServe | Speedup |
| --- | ---: | ---: | ---: |
| Common (19 bytes): `/api/users/00000000` | 128.80 ns/op | 35.98 ns/op | **3.58×** |
| Short (9 bytes) | 95.50 ns/op | 38.14 ns/op | **2.50×** |
| Medium (39 bytes) | 217.50 ns/op | 38.59 ns/op | **5.64×** |
| Long (172 bytes) | 646.00 ns/op | 39.10 ns/op | **16.52×** |

Across all four fixtures, HTTPServe used **0 B/op and 0 allocations/op**;
`http.ServeMux` used **16 B/op and 1 allocation/op**.

Measured on 2026-10-18 with Go 1.27.0 (`darwin/arm64`) on Apple M3, using one CPU
and ten alternating rounds between main (`97491b3`) and this branch (`ec85ad9`),
with one-second samples per case. The table uses this branch's samples.
All HTTPServe-versus-ServeMux timing differences were statistically
significant in `benchstat` (`p < 0.001`). These are warmed successful-route
benchmarks that include native parameter lookup and a minimal handler, excluding
request construction, network I/O, and application work. They do not measure
end-to-end HTTP throughput or compare all routing semantics.

See the [benchmark source](router_stdlib_benchmark_test.go),
[raw samples](benchmarks/trie-prefix-pr/samples/branch.txt), and
[statistical comparison](benchmarks/trie-prefix-pr/stdlib-comparison.txt).
The [comparison report](benchmarks/trie-prefix-pr/README.md) includes reproduction
commands, fixture details, and a separate comparison against HTTPServe on main.

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
