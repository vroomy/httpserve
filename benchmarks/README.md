# Router benchmarks

The repeatable suite in [router_benchmark_test.go](../router_benchmark_test.go)
compares the original route list with the compressed prefix trie. The benchmark
code and fixtures are the same for both runs.

For the implemented allocation improvements, before/after measurements, and
remaining ideas, see the [trie allocation review](trie-allocation-review.md).

## Coverage

Each benchmark runs the following matrix (162 cases total):

- URL lengths: Short, Medium, and Long.
- Route kinds: Static, Param (one named parameter), and Wildcard.
- Route counts per GET method: 1, 100, and 1,000.
- Lookups: FirstHit, LastHit, and Miss. With one route, FirstHit and LastHit
  intentionally exercise the same route.
- Entry points: `Router.Match` and `Router.ServeHTTP`.

The concrete request paths have the same length across all three route kinds:

| Length | Bytes | Segments | Request path |
| --- | ---: | ---: | --- |
| Short | 9 | 2 | `/r0000/42` |
| Medium | 39 | 7 | `/api/v1/projects/catalog/items/r0000/42` |
| Long | 172 | 18 | `/api/v1/organizations/example-corporation/workspaces/production/projects/product-catalog/environments/staging/regions/us-west/resources/collections/inventory/items/r0000/42` |

Routes share the prefix shown above and differ in their `rNNNN` segment. Static
routes end in `/42`, parameter routes in `/:id`, and wildcard routes in `/*`.
Misses request an unregistered `rNNNN` value after traversing the shared prefix.
This exposes the current list scan's dependence on registration position and
route count. URL depth and byte length increase together; this suite does not
isolate those two factors from each other.

Registration, request construction, and correctness checks happen outside the
timed loop. Before measuring, each case checks the selected handler, match result,
and parameters. `Match` retains its results in sinks. `ServeHTTP` warms the Context
pool and reuses a request and a response writer that discards output. Its handlers
only record which route ran and write a status code. It includes dispatch and
Context pooling, but excludes network I/O, response body buffering, and application
work. These are serial GET benchmarks; startup and concurrent throughput are not
measured.

## Capture and compare

Run from the repository root, on the same machine with the same Go version and
without other CPU-intensive work. The single CPU setting keeps `GOMAXPROCS`
consistent. Ten samples provide data for `benchstat` to summarize and compare.
Run the before capture against the list implementation, and the after capture
against the trie implementation. The saved before capture already records the
list implementation at the commit identified below.

```sh
go test -run '^$' -bench '^BenchmarkRouter(Match|ServeHTTP)$' \
  -benchmem -benchtime=100ms -count=10 -cpu=1 > benchmarks/router-before.txt
```

After changing the router, run the identical suite into a separate file:

```sh
go test -run '^$' -bench '^BenchmarkRouter(Match|ServeHTTP)$' \
  -benchmem -benchtime=100ms -count=10 -cpu=1 > benchmarks/router-after.txt
benchstat benchmarks/router-before.txt benchmarks/router-after.txt
```

Use `benchstat benchmarks/router-before.txt` to summarize the baseline alone.
If `benchstat` is not installed, install it with
`go install golang.org/x/perf/cmd/benchstat@latest` and ensure your Go binary
directory is on `PATH`. Neither the benchmark nor the library depends on benchstat.

For a focused comparison, select a matching subset in both captures, for example:

```sh
go test -run '^$' \
  -bench '^BenchmarkRouterServeHTTP$/(Short|Medium|Long)/Param/Routes=1000/LastHit$' \
  -benchmem -benchtime=1s -count=10 -cpu=1
```

Results include `ns/op`, `B/op`, and `allocs/op`. Compare matching case names;
the public API returns owned parameters while `ServeHTTP` reuses pooled storage,
so their allocation results can differ.

## Saved baseline

[router-before.txt](router-before.txt) contains all ten samples for the original
list-based router. [router-before-summary.txt](router-before-summary.txt) contains
the benchstat summary.

- Captured: 2026-10-13 (America/Los_Angeles).
- Router source: commit `7c36f097d2b5eb2ab6a40d078cd98a99e39c88fb`, with only
  benchmark and documentation additions in the working tree.
- Go: `go1.26.5 darwin/arm64`.
- Processor: Apple M3.
- Settings: `-benchtime=100ms -count=10 -cpu=1`; setup and warm-up excluded.
- Benchstat: `golang.org/x/perf v0.0.0-20240707193131-dc66afd55b77`.

The raw output records the processor model and operating system/architecture.
Timing differences depend on the machine and workload; these files establish a
local comparison point for the trie implementation.

For example, the median last-hit times with 1,000 parameter routes were:

| URL length | `Match` | `ServeHTTP` |
| --- | ---: | ---: |
| Short (9 bytes) | 3.644 µs/op | 4.403 µs/op |
| Medium (39 bytes) | 4.565 µs/op | 4.776 µs/op |
| Long (172 bytes) | 7.119 µs/op | 7.359 µs/op |

In these cases `Match` used 32 B/op and one allocation, and `ServeHTTP` used
16 B/op and one allocation. The full summary includes confidence intervals:
the long `Match` example had ±14% timing uncertainty, while the other examples
above had ±1%. Use the longer focused run above if a future difference is small.

## Trie implementation and validation

[route.go](../route.go) stores a compressed prefix trie for each HTTP
method. Static edges share common byte prefixes; parameter branches share their
structure while keeping each endpoint's parameter names in
[routeEndpoint.go](../routeEndpoint.go). The trie fills parameter values as a
successful traversal returns into caller-owned or pooled storage. Failed branches
do not modify capture storage, and successful matches assign names from the
selected endpoint. The old route-pattern record and second path scan have been removed;
`route` now names the trie itself.

Matching now prefers static branches, then parameters, then wildcards, with
fallback when a branch fails to complete. This replaces registration-order
precedence for overlapping routes. Duplicate route shapes retain the first
handler and parameter names. Wildcards continue to match a nonempty remainder
without returning a wildcard parameter. Registration still normalizes repeated
and trailing slashes; request paths are matched as supplied.

Validation passed:

- `go test ./...`
- `go test -race ./...`, including concurrent requests against the trie.
- `go vet ./...` and `go build ./...`.
- All 162 benchmark correctness checks.
- `go test -run '^$' -fuzz '^FuzzRouterTrie$' -fuzztime=30s -parallel=4`:
  1,080,325 executions. The fuzz test checks match existence and the selected
  route's parameters against the original route checker, allowing the new
  precedence for overlaps.

## Saved initial trie run

[router-after.txt](router-after.txt) contains the ten samples per case for the
trie. [router-after-summary.txt](router-after-summary.txt) summarizes that capture,
and [router-comparison.txt](router-comparison.txt) contains the full benchstat
before/after comparison.

- Captured: 2026-10-13 (America/Los_Angeles).
- Initial trie source: `router.go`, `route.go`, and `routeTrie.go` as captured
  on 2026-10-13, before the endpoint refactor described below,
  based on commit `7c36f097d2b5eb2ab6a40d078cd98a99e39c88fb`.
- Go, processor, benchmark fixtures, benchstat version, and measurement flags
  match the baseline. The trie capture used `GOCACHE=/tmp/httpserve-go-build`
  because the default build cache was restricted by the sandbox.

## Comparison

Both captures contain 162 cases with ten samples each. Case names and machine
metadata match, and the original baseline file is unchanged. The saved comparison
was produced with:

```sh
benchstat benchmarks/router-before.txt benchmarks/router-after.txt \
  > benchmarks/router-comparison.txt
```

Median `ServeHTTP` times for the last hit among 1,000 parameter routes:

| URL length | List | Trie | Speedup |
| --- | ---: | ---: | ---: |
| Short (9 bytes) | 4,402.5 ns/op | 102.65 ns/op | 42.89× |
| Medium (39 bytes) | 4,776 ns/op | 101.80 ns/op | 46.92× |
| Long (172 bytes) | 7,359 ns/op | 105.85 ns/op | 69.52× |

The trie removes the scan of unrelated routes. Larger route tables, last hits,
and misses benefit from sharing prefixes. It adds traversal overhead for cases
where the list previously matched immediately, including these regressions:

| `ServeHTTP` case | List | Trie | Change |
| --- | ---: | ---: | ---: |
| Short static, 1 route, first hit | 38.66 ns/op | 48.91 ns/op | +26.53% |
| Short parameter, 1,000 routes, first hit | 47.16 ns/op | 103.80 ns/op | +120.10% |

Benchstat reports statistically significant timing differences for these
examples. Bytes and allocations per operation are unchanged in every case in the
matrix. The parameter examples above still use 16 B/op and one allocation in
`ServeHTTP`.

These results favor the trie for larger route tables and requests that previously
scanned many candidates. Applications with a few routes or traffic concentrated
on the first registered route can pay additional lookup cost. Registration time,
route storage, concurrent throughput, and overlapping-pattern performance are
outside this timing suite; overlap behavior is covered by the correctness tests.

## Common user URL

[router_users_benchmark_test.go](../router_users_benchmark_test.go) adds a focused
`BenchmarkRouterUsers` suite for `/api/users/00000000`: 19 bytes, three segments,
and one eight-character parameter matched by `/api/users/:id`.

This suite covers the application's expected range with 20, 40, 50, and 100
distinct GET route patterns. Each table registers `/api/users/:id` first, in the
middle, or last; the other patterns are `/api/rNNNN/:id`, sharing the `/api/` prefix.
Different user IDs use the same route pattern. Misses request the unregistered
`/api/other/00000000`, also 19 bytes.

The measurement uses `ServeHTTP` with the same warmed Context pool and discarding
writer as the original suite. The handler reads the ID with `ctx.Param("id")`,
records which route ran, and writes a status. Setup, registration, and match,
parameter, and handler checks are excluded from the timer.

Both implementations use identical benchmark fixtures, with ten samples per case:

```sh
go test -run '^$' -bench '^BenchmarkRouterUsers$' \
  -benchmem -benchtime=200ms -count=10 -cpu=1
```

The list capture ran in an isolated temporary checkout exported from commit
`7c36f097d2b5eb2ab6a40d078cd98a99e39c88fb`, with both benchmark files copied in.
The trie capture ran in the current working tree. Both used Go 1.26.5 on Apple M3,
`GOCACHE=/tmp/httpserve-go-build`, and the flags above, on 2026-10-13.

Raw data: [router-users-before.txt](router-users-before.txt) and
[router-users-after.txt](router-users-after.txt).
The [focused benchstat comparison](router-users-comparison.txt) includes all
12 original cases (20, 50, and 100 routes) and their confidence intervals. The
40-route middle-hit capture is recorded separately below.

Median successful-request times, in ns/op (`list → trie`):

| GET route patterns | First hit | Middle hit | Last hit |
| ---: | ---: | ---: | ---: |
| 20 | 47.80 → 76.26 | 94.91 → 77.01 | 142.15 → 78.20 |
| 50 | 48.36 → 77.94 | 164.05 → 78.52 | 257.00 → 79.32 |
| 100 | 48.45 → 78.71 | 266.30 → 79.05 | 491.20 → 79.69 |

For this URL and route-count range, middle hits improved 1.23–3.37× and last hits
improved 1.82–6.16×. First hits were 59.5–62.5% slower because the list already
found the route immediately. Misses improved 2.52–9.35×. All timing differences
were statistically significant in benchstat; every case used 16 B/op and one
allocation in both implementations.

The common endpoint's position in the old list affects the gain as much as URL
length. These focused results are a closer fit for 19-byte API paths and 20–100
GET patterns than the earlier last-hit examples with 1,000 routes. Actual traffic
across endpoints and HTTP methods determines how the application benefits.

## 40 routes, middle hit

This focused capture requests `/api/users/00000000` from a table of exactly 40 GET
patterns. `/api/users/:id` is the 21st registered pattern (index 20), one of the
two central positions: the original list checks 20 other routes before matching.
It uses the same handler, ID lookup, warm-up, correctness checks, and background
patterns as the common-user-URL suite.

Captured on 2026-10-14 using Go 1.26.5 on Apple M3. Both implementations used the
same benchmark files, `GOCACHE=/tmp/httpserve-go-build`, and ten one-second
samples. The list source was verified against commit
`7c36f097d2b5eb2ab6a40d078cd98a99e39c88fb` in the isolated temporary checkout.

```sh
go test -run '^$' -bench '^BenchmarkRouterUsers$/^Routes=40$/^MiddleHit$' \
  -benchmem -benchtime=1s -count=10 -cpu=1
```

Saved data: [list](router-users-40-before.txt), [trie](router-users-40-after.txt),
and [benchstat comparison](router-users-40-comparison.txt).

| Implementation | Median `ServeHTTP` time | Bytes/op | Allocations/op |
| --- | ---: | ---: | ---: |
| List | 130.30 ns/op (±2%) | 16 | 1 |
| Trie | 77.49 ns/op (±3%) | 16 | 1 |

The trie is 1.68× faster for this case, reducing time per operation by 40.53%.
Benchstat reports a statistically significant difference with ten samples per
implementation. Allocations are unchanged. These timings include dispatch and
reading the ID parameter with a minimal handler; network I/O and application
work are excluded.

## Endpoint refactor

The preceding captures measure the initial trie, which retained a parsed route
record and checked the winning path again to extract parameters. The endpoint
implementation measured here captured values while traversing the
trie and rolled back captures when a branch failed. After choosing an endpoint, it
assigned that endpoint's parameter names without scanning the path again. The
later [allocation changes](trie-allocation-review.md#implemented-follow-up) defer
capture writes until a complete match instead.

`route` now names the trie, `routeNode` its branches, and
[routeEndpoint](../routeEndpoint.go) stores only the handler and parameter names.
The old parsed-pattern and method fields and matching helpers are removed from
production. An independent linear matcher remains in
[route_reference_test.go](../route_reference_test.go) for differential fuzzing.
Empty route registration now returns a missing-leading-slash error instead of
panicking while indexing the empty string.

Validation passed: `go test ./...`, `go test -race ./...`, `go vet ./...`, and
`go build ./...`; all 178 benchmark cases passed their correctness checks. A
30-second fuzz run completed 1,832,073 executions without failures. Tests cover
parameter rollback, wildcard fallback, shared branches with different names,
growing borrowed storage, and ownership of `Match` results.

The fresh before/after capture uses the same 40-route middle-hit case and flags
as above, on 2026-10-14, with Go 1.26.5 and Apple M3. The before source was copied
from the working tree into an isolated temporary snapshot before the refactor;
the after capture uses the renamed trie and endpoint implementation. Both runs
use identical benchmark fixtures and `GOCACHE=/tmp/httpserve-go-build`.

| Implementation | Median `ServeHTTP` time | Bytes/op | Allocations/op |
| --- | ---: | ---: | ---: |
| Trie with second path scan | 79.90 ns/op (±3%) | 16 | 1 |
| Trie with parameter capture during traversal | 71.24 ns/op (±2%) | 16 | 1 |

This refactor reduced time by 10.84% (1.12× faster) in the focused case.
Benchstat reports a statistically significant difference over ten one-second
samples per implementation. Allocations are unchanged.

Saved data: [before](router-users-40-endpoint-before.txt),
[after](router-users-40-endpoint-after.txt), and
[benchstat comparison](router-users-40-endpoint-comparison.txt).

## Standard library comparison

[router_stdlib_benchmark_test.go](../router_stdlib_benchmark_test.go) compares the
current trie router with Go's `http.ServeMux`. Each has exactly 40 GET route
patterns, with the target at registration index 20 (21st in the table). Four
request paths are measured:

| Fixture | Bytes | Request path |
| --- | ---: | --- |
| Common | 19 | `/api/users/00000000` |
| Short | 9 | `/r0020/42` |
| Medium | 39 | `/api/v1/projects/catalog/items/r0020/42` |
| Long | 172 | `/api/v1/organizations/example-corporation/workspaces/production/projects/product-catalog/environments/staging/regions/us-west/resources/collections/inventory/items/r0020/42` |

The trie registers `prefix/resource/:id`; ServeMux registers the equivalent
`GET prefix/resource/{id}`. Both handlers record the selected route, read the ID
using their native API (`Context.Param` or `Request.PathValue`), and write a 204
status. The benchmark checks the selected handler, parameter value, and status
before timing and after the loop.

Both routers dispatch through `http.Handler`, reuse a prepared request and a
discarding response writer, and warm up before measurement. Local Go 1.26.5
source confirms ServeMux performs routing again on each `ServeHTTP` call;
reusing the request does not skip matching. Request construction, route
registration, network I/O, response body buffering, and application work are
excluded. The trie measurement includes its normal Context pooling and panic
recovery. ServeMux performs its normal host, method, and path processing.

These are successful GET cases with canonical ASCII paths and one named
parameter. They do not compare redirects, escaped paths, HEAD behavior, misses,
or startup costs; the routers' behavior differs in some of those areas.

### Before allocation improvements (2026-10-14)

Captured on 2026-10-14 with Go `go1.26.5 darwin/arm64` on Apple M3, using the
current working tree and the installed standard-library implementation. The
module declares Go 1.25, and `GODEBUG` is unset, so the modern ServeMux wildcard
syntax is used. All eight correctness checks passed before the timed capture.

```sh
GOCACHE=/tmp/httpserve-go-build go test -run '^$' -bench '^BenchmarkRouterStdlib$' \
  -benchmem -benchtime=1s -count=10 -cpu=1 > benchmarks/router-stdlib.txt
benchstat -row /URL -col /Router benchmarks/router-stdlib.txt \
  > benchmarks/router-stdlib-comparison.txt
```

Saved data: [raw samples](router-stdlib.txt) and
[benchstat comparison](router-stdlib-comparison.txt).

Median successful-request times over ten one-second samples per case:

| Fixture | ServeMux | Trie | Trie speedup |
| --- | ---: | ---: | ---: |
| Common (19 bytes) | 127.10 ns/op | 70.45 ns/op | 1.80× |
| Short (9 bytes) | 97.70 ns/op | 83.22 ns/op | 1.17× |
| Medium (39 bytes) | 221.10 ns/op | 81.53 ns/op | 2.71× |
| Long (172 bytes) | 640.30 ns/op | 82.94 ns/op | 7.72× |

All timing differences were statistically significant in benchstat. Both routers
used 16 B/op and one allocation in every case. For the common URL, the trie used
44.57% less time. The advantage increases in these deeper URL fixtures because
the trie combines shared static segments into compressed prefixes; ServeMux
processes individual segments for matching and native parameter lookup.

The short fixture's numeric resource name shares prefixes with the other 39
resources, while the common fixture's `users` resource branches off earlier.
That changes the trie's path through its nodes and explains why the 19-byte
common case can be cheaper than the 9-byte short case. URL length alone does not
determine lookup cost.

`go test ./...`, `go vet ./...`, and `go build ./...` passed after adding the
comparison suite. No production code changed for this comparison.

### After allocation improvements (2026-10-15)

This capture measures the working tree with the
[implemented allocation changes](trie-allocation-review.md#implemented-follow-up),
using the unchanged `BenchmarkRouterStdlib` fixtures above. Both routers ran in
the same invocation with Go `go1.26.5 darwin/arm64` on Apple M3, `GODEBUG` unset,
one CPU, and ten one-second samples per case.

```sh
GOCACHE=/tmp/httpserve-go-build go test -run '^$' -bench '^BenchmarkRouterStdlib$' \
  -benchmem -benchtime=1s -count=10 -cpu=1 > benchmarks/router-stdlib-allocations.txt
benchstat -row /URL -col /Router benchmarks/router-stdlib-allocations.txt \
  > benchmarks/router-stdlib-allocations-comparison.txt
```

Saved data: [raw samples](router-stdlib-allocations.txt) and
[benchstat comparison](router-stdlib-allocations-comparison.txt).
Median dispatch times across ten samples per case:

| Fixture | ServeMux | Trie | Trie speedup |
| --- | ---: | ---: | ---: |
| Common (19 bytes) | 129.90 ns/op | 47.32 ns/op | **2.75×** |
| Short (9 bytes) | 99.52 ns/op | 58.31 ns/op | **1.71×** |
| Medium (39 bytes) | 224.95 ns/op | 58.00 ns/op | **3.88×** |
| Long (172 bytes) | 645.45 ns/op | 58.84 ns/op | **10.97×** |

The trie used **0 B/op and 0 allocs/op** in every sample. ServeMux used
**16 B/op and 1 alloc/op** in every sample. Benchstat reports statistically
significant timing differences for all four fixtures (`p < 0.001`, ten samples
per router), with the common user URL taking 63.57% less dispatch time.

All eight handler, parameter, and status correctness checks passed. These results
include warmed Context pooling, panic recovery, and native parameter lookup, but
exclude request creation and network/application work. They establish an
advantage for these successful one-parameter GET fixtures; they do not establish
the same speedup for complete HTTP requests, other route shapes, or the additional
path and method behavior supported by ServeMux. No production or benchmark source
changed for this capture.
