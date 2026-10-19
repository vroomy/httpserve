# Apex &emsp; [![GoDoc][GoDoc Badge]][GoDoc URL] ![Coverage] [![Lint][Lint Badge]][Lint URL] [![MIT licensed][License Badge]][License URL]

[GoDoc Badge]: https://pkg.go.dev/badge/github.com/vroomy/apex
[GoDoc URL]: https://pkg.go.dev/github.com/vroomy/apex
[Coverage]: https://img.shields.io/badge/coverage-64.6%25-yellow
[License Badge]: https://img.shields.io/badge/license-MIT-blue.svg
[License URL]: https://github.com/vroomy/apex/blob/main/LICENSE
[Lint Badge]: https://github.com/vroomy/apex/actions/workflows/lint.yaml/badge.svg
[Lint URL]: https://github.com/vroomy/apex/actions/workflows/lint.yaml

![Apex billboard](billboard.webp)

**High-performance HTTP for Go.**

Apex is the HTTP library in the Vroomy ecosystem, with zero-allocation warmed
routing and SIMD-accelerated matching on ARM64 and AMD64. Use it on its own or as
an HTTP foundation for your application.

## Features

- Static, named-parameter, and wildcard routing with predictable precedence
- Middleware and route groups
- Pooled request contexts and reusable parameter storage
- Targeted NEON and SSE2 routing searches, with a portable fallback
- HTTP/HTTPS serving, HTTP/2 support, and automatic TLS certificates

The zero-allocation claim applies to warmed routing with sufficient reusable
storage. Response generation, application work, and cold pools can still allocate.
See [Routing](#routing) for the implementation details and
[Benchmarks](#benchmarks) for measured results and their scope.

## Quickstart

Requires **Go 1.27 or newer**. From this checkout:

```sh
go run ./example/functions
```

Then, in another terminal:

```sh
curl http://localhost:8080/ping
# pong
```

Stop the server with Ctrl-C. The [function example](example/functions/main.go)
contains the complete program below. A [method-based example](example/methods)
is also available with `go run ./example/methods`.

```go
package main

import (
	"log"

	"github.com/vroomy/apex"
)

func main() {
	var (
		engine *apex.Engine
		err    error
	)

	engine = apex.New()
	defer engine.Close()

	if err = engine.GET("/ping", func(ctx *apex.Context) {
		ctx.WriteString(200, "text/plain", "pong")
	}); err != nil {
		log.Fatal(err)
	}

	engine.Set404(func(ctx *apex.Context) {
		ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
	})

	if err = engine.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
```

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

For reproducible comparisons between Apex and Go's `http.ServeMux`, see
[the router benchmark guide](benchmarks/README.md#prefix-branch-versus-main-and-stdlib-2026-10-18).
It includes fixture details, allocation measurements, and commands for comparing
the routers with `benchstat`.

### Standard library comparison

Comparison with 40 GET routes and one named parameter per route. Timings are
median dispatch times; speedup is `http.ServeMux` time divided by Apex time:

| URL fixture | `http.ServeMux` | Apex | Speedup |
| --- | ---: | ---: | ---: |
| Common (19 bytes): `/api/users/00000000` | 128.80 ns/op | 35.98 ns/op | **3.58×** |
| Short (9 bytes) | 95.50 ns/op | 38.14 ns/op | **2.50×** |
| Medium (39 bytes) | 217.50 ns/op | 38.59 ns/op | **5.64×** |
| Long (172 bytes) | 646.00 ns/op | 39.10 ns/op | **16.52×** |

Across all four fixtures, Apex used **0 B/op and 0 allocations/op**;
`http.ServeMux` used **16 B/op and 1 allocation/op**.

Measured on 2026-10-18 with Go 1.27.0 (`darwin/arm64`) on Apple M3, using one CPU
and ten alternating rounds between main (`97491b3`) and this branch (`ec85ad9`),
with one-second samples per case. The table uses this branch's samples.
All Apex-versus-ServeMux timing differences were statistically
significant in `benchstat` (`p < 0.001`). These are warmed successful-route
benchmarks that include native parameter lookup and a minimal handler, excluding
request construction, network I/O, and application work. They do not measure
end-to-end HTTP throughput or compare all routing semantics.

See the [benchmark source](router_stdlib_benchmark_test.go),
[raw samples](benchmarks/trie-prefix-pr/samples/branch.txt), and
[statistical comparison](benchmarks/trie-prefix-pr/stdlib-comparison.txt).
The [comparison report](benchmarks/trie-prefix-pr/README.md) includes reproduction
commands, fixture details, and a separate comparison against Apex on main.

## Migrating from httpserve

The module path is now `github.com/vroomy/apex`, and the root package is `apex`.
Update imports and package qualifiers:

| Before | After |
| --- | --- |
| `github.com/vroomy/httpserve` | `github.com/vroomy/apex` |
| `github.com/vroomy/httpserve/form` | `github.com/vroomy/apex/form` |
| `httpserve.New()` | `apex.New()` |
| `*httpserve.Serve` | `*apex.Engine` |
| `*httpserve.Context` | `*apex.Context` |

`Serve` is now `Engine`, and `New()` returns `*Engine`. Its methods and behavior
are unchanged. An explicit import alias can preserve existing package qualifiers
while migrating the type name:

```go
import httpserve "github.com/vroomy/apex"
```

With this alias, change explicit `*httpserve.Serve` declarations to
`*httpserve.Engine`.

Use an Apex version or commit whose `go.mod` declares the new module path, then
run `go mod tidy`. Tags from before the rename still declare the old path.
Migrate Vroomy and application plugins together: Go treats types imported from
the old and new module paths as distinct, even when their definitions match.

For maintainers, publish the repository as `github.com/vroomy/apex` with the
updated `go.mod` before updating downstream dependencies. A repository redirect
alone does not change module declarations in existing tags.

Historical [benchmark reports](benchmarks/README.md) retain their original
HTTPServe names and raw samples; the tables above describe the same implementation
under the Apex name.

## Development

```sh
go test ./...
go test -tags purego ./...
go vet ./...
go build ./...
```
