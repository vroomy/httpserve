# Trie efficiency and allocation review

Initially reviewed `implement-trie` at `4505171` against `main` (`7c36f09`) on
2026-10-15. The implementation follow-up below applies the request allocation
and cleanup changes, plus a smaller registration improvement. The original
measurements and findings remain below it as a record of the reviewed baseline;
their source line numbers refer to `4505171`.

## Implemented follow-up

The current implementation removes the escaping callback and defers parameter
writes until traversal reaches a winning endpoint. It reserves exactly the
successful route's required space, then fills values as recursive calls return.
Failed branches never touch capture storage. This goes beyond lazy allocation
on the first attempted parameter: deep misses and parameter-free wildcard
fallbacks also allocate nothing.

`Router.Match` continues to return independent owned results. Parameter-free
results now use a nil slice. Global/per-method maximum counts are unnecessary:
other routes and ignored wildcard suffixes cannot inflate the selected result.
`Router.MatchInto(method, path, params)` returns `(handler, params, ok)` and reuses
caller-provided storage. It clears the supplied slice's existing elements,
starts at length zero, and returns updated storage for the next call. It does
not retain the caller's buffer in the router; callers must give concurrent calls
separate buffers and copy results they need to preserve across reuse.

Context release now clears parameter and hook references across their backing
arrays and empties the storage map before pooling. This also handles slices
shortened by a handler. Existing capacities remain reusable. Error callbacks
retain custom handling and nil-callback default logging; panic recovery remains
in place.

Registration now scans segments with `strings.Cut` rather than constructing a
split slice. Endpoint parameter names are allocated once at the effective count,
only when adding a new endpoint. Slash normalization, wildcard tails, duplicate
precedence, and parameter names are unchanged.

[Implemented benchmark output](router-allocations-implemented.txt) uses the same
Go version, machine, six-route fixture, and three 100 ms samples as the baseline,
plus nine new `MatchInto` cases. Allocation counts agree across all samples:

| Case | Baseline B/op, allocs/op | Implemented B/op, allocs/op |
| --- | ---: | ---: |
| Warmed `ServeHTTP`, all nine cases | 16, 1 | **0, 0** |
| Owned `Match`, static/wildcard/early or deep miss/empty method/fallback | 160, 1 | **0, 0** |
| Owned `Match`, one parameter | 160, 1 | **32, 1** |
| Owned `Match`, five parameters | 160, 1 | 160, 1 |
| Borrowed capacity four, five-parameter hit | 256, 1 | **160, 1** |
| Borrowed capacity four, deep miss or wildcard fallback | 256, 1 | **0, 0** |
| `MatchInto`, sufficient reusable capacity, all nine cases | Not available | **0, 0** |
| Register 40 static routes | 10,088, 272 | **8,168, 232** |
| Register 40 one-parameter routes | 15,848, 392 | **13,288, 352** |

An unsupported method and internal lookup with sufficient capacity already had
zero allocations. The cold Context constructor remains unchanged: pools can
still allocate fresh Contexts and buffers, and a successful route may grow a
buffer. Growth reserves the selected result's size, so progressively deeper
successful requests can require several growth steps before reaching sufficient
capacity. There is no growth for unsuccessful attempted branches.

The exploratory suggestions for inline Context storage, node arenas, separate
name/value storage, and iterative traversal remain future work. They were not
needed to reach zero warmed dispatch allocations and need separate measurements
of retained memory and broader workloads. Timing comparisons from these three
short samples remain directional; no statistically established speedup is claimed.
For example, the five-parameter owned `Match` median was 150.0 ns/op versus
142.8 ns/op in the baseline, with unchanged allocations. The change does not
establish a latency improvement for every route even though it removes wasted
allocation on other paths.

The separate [updated standard-library comparison](README.md#after-allocation-improvements-2026-10-15)
uses ten one-second samples to compare the implemented router with ServeMux.

Regression coverage checks `MatchInto` ownership and reuse, exact owned-result
sizing, zero allocations on misses, untouched borrowed storage after failed
traversal, callback behavior, cleanup on normal and panic exits, and parser
normalization. The existing route matrix and fuzz test also exercise `MatchInto`.

Implementation validation passed:

- `go test -count=1 ./...`, `go test -race ./...`, `go vet ./...`, and
  `go build ./...` (the HTTP integration tests ran outside the sandbox).
- All 45 allocation lookup cases and both registration cases, three samples each.
- All 186 existing routing/users/standard-library benchmark correctness checks
  using `-benchtime=1x`; those runs are correctness checks, not timing evidence.
- `go test -run '^$' -fuzz '^FuzzRouterTrie$' -fuzztime=30s -parallel=4`:
  2,008,072 executions.
- Formatting, whitespace checks, and relative documentation links.

## Measurements

Environment: Go 1.26.5, darwin/arm64, Apple M3, `-cpu=1`, three 100 ms samples.
[Benchmark source](../router_allocations_benchmark_test.go),
[baseline output](router-allocations-baseline.txt), and
[callback prototype output](router-allocations-callback-prototype.txt) are saved.
Allocation counts were consistent across samples. Timing medians below are
directional measurements, not statistically established speedups.

The diagnostic table mixes six GET patterns: `/health`, `/users/:id`, `/files/*`,
`/deep/:a/:b/:c/:d/:e/end`, `/fallback/:a/:b/:c/:d/:e/end`, and `/fallback/*`.
The maximum parameter count is five. Deep misses and wildcard fallback consume
five parameters before rejecting the parameter branch. HEAD is supported but has
no routes in this table; PATCH is unsupported.

| Entry point / case | B/op | Allocs/op |
| --- | ---: | ---: |
| Public `Match`: static, one/five parameters, wildcard, misses, empty HEAD table | 160 | 1 |
| Public `Match`: unsupported PATCH | 0 | 0 |
| Internal `match`: sufficient reusable capacity, every case | 0 | 0 |
| Internal `match`: four-slot buffer, five-parameter hit/miss/fallback | 256 | 1 |
| Warmed `ServeHTTP`: every case, baseline | 16 | 1 |
| Warmed `ServeHTTP`: every case, callback prototype | 0 | 0 |

`BorrowedCap4` deliberately supplies the original four-slot backing array on
each iteration. It measures growth past the pool's initial capacity, excluding
allocation of the original buffer and Context. Normal reuse retains the grown
array and does not pay this cost on every request. `BorrowedWarm` preallocates
five slots. Setup, route registration, request construction, and correctness
checks are outside the lookup timers. Handlers only record route selection; the
not-found handler is also replaced. These are routing measurements, not the
cost of network I/O, the default 404 response, or application handlers.

## 1. Remove the per-request bound method allocation

Source: [`Router.ServeHTTP`, router.go:160](../router.go#L160).

`ctx.errorFn = r.onError` constructs a method value that captures `r` and escapes
when stored on a pooled Context. Compiler escape analysis explicitly reports
`r.onError escapes to heap`. This is the remaining 16-byte allocation even for
static routes, misses, and unsupported methods. It predates the trie change.
The recovery defer does **not** escape in this build; removing panic recovery
would target the wrong cost.

Copy the already configured `r.errorFn` into the Context. When it is nil, use a
package-level function implementing the existing default logging. Assigning an
existing function value does not create another closure. The
[exact experimental patch](router-error-callback-prototype.patch) makes only
this dispatch change and adds the fallback function. It ran through a Go source
overlay. This historical patch targets `4505171`; the implemented follow-up
uses the same callback selection with its fallback function in `utils.go`.

| Warmed dispatch case | Baseline median ns/op | Prototype median ns/op |
| --- | ---: | ---: |
| Static | 55.77 | 34.12 |
| One parameter | 72.61 | 44.00 |
| Five parameters | 142.80 | 108.80 |
| Failed parameter branch, wildcard fallback | 150.60 | 121.00 |

Preserve custom error delivery, nil-callback default logging, panic recovery,
and Context cleanup. This proposal follows the documented requirement to
configure handlers before serving. Do not lazily initialize a shared cached
callback in `ServeHTTP`, which would introduce a write on concurrent requests.
A callback that the application itself allocates while handling an error is
outside these dispatch measurements.

## 2. Stop sizing every public lookup for the largest route

Source: [`Router.Match`, router.go:57](../router.go#L57),
[`Router.Handle`, router.go:118](../router.go#L118), and
[`route.add`, route.go:25](../route.go#L25).

`Match` eagerly calls `make(Params, 0, r.maxParams)` before knowing whether any
parameters are needed. On this architecture a `Param` occupies 32 bytes. A
five-parameter route therefore makes `/health`, `/files/icon.svg`, an early
miss, and even an empty HEAD table each allocate 160 bytes. The maximum is
shared across methods, so a deep POST route would also inflate GET results.
This policy predates the trie, but now dominates many public lookup timings.

The maximum also counts names after a wildcard, although insertion stops at
the wildcard. A diagnostic registering only `/files/*/:ignored/:alsoIgnored`
returned zero parameters with capacity two and set `maxParams` to two. Those
ignored names should not influence allocation sizing.

Suggested order:

1. Return immediately when the selected method's trie is empty. Track effective
   parameter maxima per method and stop counting at the first wildcard.
2. Offer an explicit caller-buffer API, such as `MatchInto`, for repeated public
   lookups. It can reuse the existing borrowed-storage implementation. Specify
   reset/append behavior, ownership, misses, and reuse across concurrent calls.
   With sufficient capacity, this is the demonstrated path to zero allocations.
3. Consider lazy capture allocation for the owned `Match` API. Reserving an
   appropriate capacity only on the first attempted capture avoids allocation
   for static/wildcard hits and misses that never enter a parameter branch.
   Late misses and fallback may still allocate. Measure before changing it.

Do not simply replace the eager allocation with a nil slice and assume it is
better for every route: successive appends can allocate several growing arrays
on deeper matches. Do not return pooled storage from the existing `Match` API;
its documented results belong to the caller and must survive later lookups.
Owned nonempty results still need independent backing storage with this API.

A local scratch array is also not automatically stack storage. The current
`find(url, *Params)` escape summary leaks the slice's backing storage through
the append-and-assignment operation. A one-off benchmark declaring `[4]Param`
inside each lookup and passing its slice to `match` still used **128 B / 1
allocation** per lookup. A scratch-and-copy approach needs a different capture
interface and verified escape analysis to avoid adding another allocation.

## 3. Clear retained references, including abandoned captures

Source: [`routeNode.find`, routeNode.go:85–92](../routeNode.go#L85) and
[`releaseContext` / `acquireContext`, context.go:33–56](../context.go#L33).

Captures are substrings of the request path. This avoids string copies, but a
small captured value can keep the whole path's backing storage alive.
Backtracking shortens the parameter slice without clearing abandoned slots.
Context release clears the request and writer, but leaves Params, hooks, and
storage entries holding references. The next acquire only shortens slices and
deletes map entries; shortening does not erase the slice's backing array.

A diagnostic using `/users/:id/details` plus `/users/*` and requesting
`/users/<1024-byte-id>/other` ended with `len(Params) == 0`, capacity four, and
the full 1,024-byte value still in slot zero **after Context release**. This is
retained memory, so `allocs/op == 0` cannot rule it out. The pool cleanup issue
predates this branch; failed trie captures add another way to leave invisible
references behind.

For a straightforward fix, clear the entire owned parameter backing buffer at
release (`clear(c.Params[:cap(c.Params)])`) before resetting its length. Clearing
only `len(c.Params)` misses rolled-back captures. Clear hook references and map
entries at release as well so idle pooled Contexts do not retain request data.
Alternatively, clear discarded captures before every rollback, clear successful
captures on release, and maintain that invariant on all paths. A tracked
high-water mark can reduce clearing work but adds bookkeeping.

Measure the CPU cost of clearing against reduced live heap. Retain reasonable
buffer capacity for reuse; consider discarding unusually large buffers only
with workload evidence. Clearing values does not free the backing array, and
discarding it trades retention for future allocations. Pool retention is not
necessarily permanent: entries can be reused or removed by the runtime.

## 4. Treat cold pools and deep backtracking separately

Source: [`ctxPool.New`, context.go:25–30](../context.go#L25) and
[`routeNode.find`, routeNode.go:76–92](../routeNode.go#L76).

A fresh Context allocates its own struct and a four-element Params array.
A one-off benchmark of `ctxPool.New()` measured **240 B / 2 allocations** here,
before any callback or capture growth. The fifth capture grows the four-slot
buffer to eight slots, measured as another **256 B / 1 allocation**. That growth
also happens on deep misses and a wildcard result with no returned parameters.
Capacity must cover attempted branches, not only the winning endpoint.

An inline small Params array inside Context could combine the initial objects
into one allocation. It also makes every Context larger, including contexts
serving only static routes, and the inline storage remains present after growth.
Choose its size from observed capture depth; do not increase it globally just
to make an already warmed benchmark look better. Per-router pools sized from
effective capture depth are another option, but introduce more pools and
retention to manage.

`sync.Pool` may discard entries; serial warm-up is not a guarantee of zero
allocations after GC or under increased concurrency. Before promising broader
zero-allocation behavior, add separate measurements for cold Context creation,
parallel requests with a writer per worker, mixed shallow/deep requests, and
reuse after GC. Do not put forced GC inside a normal per-request timing loop.

## 5. Reduce registration allocations if startup or table size matters

Source: [`getParts`, utils.go:22–58](../utils.go#L22),
[`route.add`, route.go:10–38](../route.go#L10), and
[`routeNode.insertStatic`, routeNode.go:18–50](../routeNode.go#L18).

The new registration benchmark builds a fresh 40-route table per operation.
Pattern formatting and handler creation are outside the timer; router
construction is included. These numbers describe total allocated bytes during
construction, **not** the final live size of the trie.

| Table | Median ns/op | B/op | Allocs/op |
| --- | ---: | ---: | ---: |
| 40 static routes | 7,631 | 10,088 | 272 |
| 40 one-parameter routes | 10,465 | 15,848 | 392 |

Possible changes, in increasing implementation cost:

- Replace `strings.Split` plus temporary byte-buffer construction with a scanner
  that preserves the current repeated/trailing-slash normalization. A fast path
  for already normalized static spans can borrow substrings. Borrowing saves
  copies but can retain a large source string; avoid accidental retention of
  whole configuration buffers.
- Pre-size endpoint parameter-name slices using the effective count before the
  wildcard, avoiding repeated append growth. Keep duplicate-pattern behavior.
- Reduce repeated `indices += ...` copies and child-slice growth during building.
  A builder/finalization phase could compact them once registration finishes.
- Consider indexed contiguous node storage only if profiles show node-object
  overhead matters. Here a node's struct payload is 80 bytes and an endpoint's
  is 32 bytes, excluding child arrays and strings. Avoid retaining pointers into
  a growable slice and assuming later appends preserve one coherent arena.

Embedding the full endpoint in every node saves some allocations but enlarges
nonterminal nodes. Likewise, a map or a 256-entry child table may add memory to
the common low-fanout case. Compare retained bytes and lookup time, not just
the number of startup allocations. No registration optimization was prototyped.

## 6. Further lookup speed needs broader workloads

The current `strings.HasPrefix`, `strings.IndexByte`, and string slicing do not
allocate. The method array, shared prefixes, endpoint-owned names, and reuse of
grown capture storage are useful properties to preserve.

The remaining CPU candidates are pointer chasing/recursive calls on static
chains, child-index scans, writing each capture and then its key, and repeated
traversal when static or parameter branches fail. Measure single-child fast
paths or iterative static descent before adding complexity. Any iterative
version must still restore the correct fallback state; it must not turn an
allocation-free recursive traversal into an allocating explicit stack.

The existing large table benchmarks mainly vary static numeric prefixes and
have one parameter. The diagnostics here add five-parameter failure and wildcard
fallback, but still do not establish worst-case overlap performance. Add varied
fanout and nested static/parameter ambiguity, especially late misses. Lookup
work depends on the branches actually visited, not just URL length; backtracking
can revisit the same remaining URL at several competing branches. Avoid claiming
unconditional route-count-independent lookup cost from the current matrix.

Separating capture values from endpoint names could reduce per-capture storage
and writes, but public `Context.Params` exposes a mutable slice of key/value
pairs. Treat that as a larger API/design change, after the callback and allocation
policy changes have been measured.

## Reproduce and validate

```sh
GOCACHE=/tmp/httpserve-go-build go test -run '^$' \
  -bench '^BenchmarkRouter(Allocations|Registration)$' \
  -benchmem -benchtime=100ms -count=3 -cpu=1

# Inspect compiler allocation decisions without disabling optimizations.
GOCACHE=/tmp/httpserve-go-build go test -run '^$' \
  -gcflags='github.com/vroomy/httpserve=-m=2' . 2> /tmp/httpserve-escape.txt
rg 'r.onError escapes|leaking param content: params|make\(Params' \
  /tmp/httpserve-escape.txt
```

To reproduce the original prototype in a disposable checkout at `4505171`, use
the original benchmark fixture (without `MatchInto`), apply the saved patch, and
run the same benchmark restricted to
`'^BenchmarkRouterAllocations$/.*/^ServeHTTP$'`. For a performance decision, use
longer runs with at least ten samples and compare with `benchstat`; do not treat
the three short samples as a confidence interval.

Completed checks:

- `go test -count=1 ./...` and `go test -race ./...`.
- `go vet ./...` and `go build ./...`.
- All 36 diagnostic lookup benchmarks and both registration benchmarks, with
  three samples each; all nine prototype dispatch cases, also three samples each.
- The full test suite against the callback overlay, including temporary checks
  for custom error delivery, default logging, and resetting the callback to nil.
- Temporary diagnostics for retained capture references, ignored wildcard-tail
  sizing, cold Context allocation, and escaping local scratch storage.
- `gofmt`, whitespace checks, relative documentation links, and a dry-run check
  that the prototype patch applies to the reviewed source.

The integration tests needed execution outside the sandbox to bind port 8080;
their unrestricted reruns passed. The allocation attribution above uses compiler
escape analysis plus the isolated callback experiment. A memory profile was
captured, but this environment lacks the
`pprof` tool, so no profile-based attribution is claimed. HTTP end-to-end,
parallel throughput, and live-heap improvements from the proposed cleanup
remain unmeasured. The repository instructions reference Vroomy development
guides and `examples/hello` that are absent from this HTTPServe checkout; this
review uses the actual source and benchmark fixtures present here.
