# ARM64 routing byte searches

## Implementation

The routing profile of the long `BenchmarkRouterStdlib` fixture attributed about
21% of CPU samples to `internal/bytealg.IndexByteString` and 12% to
`runtime.memequal` on this machine. Both already use architecture-specific code.
The selected change targets the short byte searches used to select trie children
and find parameter delimiters, without changing prefix comparisons or routing
semantics.

[routeIndexByte_arm64.go](../routeIndexByte_arm64.go) inlines empty-string and
first-byte checks into the matcher. Remaining searches call
[routeIndexByte_arm64.s](../routeIndexByte_arm64.s). It uses scalar loads below
four bytes and NEON `VCMEQ` to compare larger inputs, then finds the first matching
lane. Inputs of 4–15 bytes use bounded loads of their first and last 4 or 8 bytes.
Inputs of 16–31 bytes use up to two overlapping 16-byte loads. Production
searches of 32 bytes or more retain `strings.IndexByte`; its streaming search
beat the custom routine on long inputs and wide child tables. Every custom load
stays inside the string: no padding, adjacent-memory access, or allocation is required.
Arbitrary bytes, including NUL, are supported. See the
[Go assembler reference](https://go.dev/doc/asm) for the assembly syntax.

The inlined shortcuts are essential. An earlier dispatcher that could not inline
improved the four headline HTTP fixtures but regressed some large static/wildcard
lookups. The final dispatcher bypasses the assembly call for empty strings and
first-byte hits, and avoids vector setup for tiny inputs. The control below uses
the same shortcuts directly in the matcher and calls `strings.IndexByte` for the
remaining work. The final size dispatch stays at the call sites so it does not
prevent the short-search wrapper from inlining. Keeping the control explicitly inline avoids attributing Go
function-call overhead to SIMD. Differences versus that control measure the
custom search routine as a whole, including scalar assembly and call conventions,
not just the vector instructions.

[routeIndexByte_generic.go](../routeIndexByte_generic.go) uses `strings.IndexByte`
outside ARM64/AMD64 and with `-tags purego`. AMD64 now has a separate
[SSE2 implementation and validation report](router-simd-amd64.md); the measurements
below describe the ARM64 implementation. That tag disables this package's
custom assembly; the standard library can still use its own SIMD. No experimental
Go features, CPU detection, new dependencies, or public APIs are required.

## Measurement

Captured on 2026-10-17 (America/Los_Angeles), Apple M3, Go 1.27.0,
`darwin/arm64`, `GOARM64=v8.0`, `GOEXPERIMENT` and `GODEBUG` unset.
All binaries use `GOCACHE=/tmp/httpserve-go-build`. Each focused fixture has ten samples
of 100 ms with `-cpu=1`. Three binaries run serially in alternating order each
round to reduce order and temperature effects:

- **Before:** commit `f8c7f8fbfed9505a8ad3434c27041e1ff6da77eb`, plus the new
  benchmark fixtures, with no production changes.
- **Fast paths:** the same baseline plus the
  [control patch](router-simd-fastpaths.patch), which handles empty strings and
  first-byte hits inline and uses `strings.IndexByte` for everything else.
- **After:** the ARM64 implementation in this change, with the same fixtures.

The existing `BenchmarkRouterStdlib` covers four 40-route tables. New
[benchmarks](../router_simd_benchmark_test.go) cover terminal and nested parameters
of 2, 8, 16, 32, 64, 256, and 1,024 bytes, plus rotating requests through 4, 16,
and 64 sibling edges. Each checks the selected handler or parameter and HTTP
status before and after timing. Requests and route registration are untimed;
dispatch includes warmed Context pooling and a minimal handler. These are
in-process routing measurements, excluding network I/O and application work.

Raw samples: [before](router-simd-before.txt),
[fast paths](router-simd-fastpaths.txt), [after](router-simd-after.txt).

## Results

Median dispatch times for the existing 40-route fixtures:

| URL fixture | Before | Fast paths only | After | Time saved vs before |
| --- | ---: | ---: | ---: | ---: |
| Common (19 bytes) | 47.76 ns | 38.68 ns | 39.34 ns | 17.62% |
| Short (9 bytes) | 57.92 ns | 41.96 ns | 42.31 ns | 26.96% |
| Medium (39 bytes) | 57.83 ns | 42.50 ns | 42.42 ns | 26.64% |
| Long (172 bytes) | 57.92 ns | 42.57 ns | 42.59 ns | 26.48% |

All four before/after differences have `p < 0.001`, with ten samples per case.
Every HTTPServe case in the focused capture remains at **0 B/op and 0 allocs/op**.
The unchanged ServeMux Common case varied by +1.60%; the other ServeMux changes
were not statistically significant. Small timing differences should be treated
in that context.

**Most of the headline improvement comes from the inlined checks.** Compared with
the equally inlined standard-library control, the custom search saves 6.00% in
the rotating 16-child fixture and 2.28% in the nested eight-byte parameter fixture.
It does not significantly improve Short, Medium, or Long in the table above, and
Common costs 1.72% (0.66 ns) more than the control. Some parameter fixtures also
favor the control, by up to 3.11%. SIMD is therefore a targeted experiment with
workload-dependent benefits, not evidence that custom assembly is universally
better than Go's already optimized routines.

Before/after reductions across the terminal/nested parameter fixtures range from
7.79% to 16.31%. Long parameter scans themselves still use the standard library;
the surrounding short trie searches account for their improvement. The rotating
64-child fixture had high timing uncertainty and no significant before/after
difference (`p = 0.075`). Its median should not be treated as a reliable speedup.

Full statistical output: [before/after](router-simd-comparison.txt) and
[fast paths/custom search](router-simd-neon-comparison.txt). Benchstat version:
`golang.org/x/perf v0.0.0-20240707193131-dc66afd55b77`.

### Broad routing matrix

The unchanged `BenchmarkRouterMatch` and `BenchmarkRouterServeHTTP` suites add
162 cases spanning three URL lengths, static/parameter/wildcard routes, 1/100/1,000
registered routes, and first hits, last hits, and misses. This screening capture
uses six 30 ms samples per case, again alternating before/after order and running
serially with `-cpu=1`.

The geometric mean time fell **14.99%**, from 29.47 to 25.05 ns/op. Allocations
were unchanged throughout. Two direct `Match` cases regressed significantly:

| Long URL, 1,000 routes, last hit | Before | After | Change |
| --- | ---: | ---: | ---: |
| Static | 35.20 ns | 36.56 ns | +3.85% |
| Wildcard | 37.73 ns | 40.17 ns | +6.47% |

Both have `p = 0.002`; no `ServeHTTP` timing regression was significant in this
capture. The shorter samples are useful for finding tradeoffs across the matrix,
not for claiming universal optimality. Applications dominated by those direct
lookups should compare against `-tags purego` using their own route tables.

Saved data: [before](router-simd-matrix-before.txt),
[after](router-simd-matrix-after.txt), and
[statistical comparison](router-simd-matrix-comparison.txt). Reproduce this matrix
with `-bench '^BenchmarkRouter(Match|ServeHTTP)$' -benchtime=30ms -count=6 -cpu=1`.
All results here measure in-process routing on this machine and toolchain;
they do not measure end-to-end HTTP throughput.

## Reproduction

For a quick comparison on ARM64, run the same suite using the fallback and default
builds. The fallback restores the original string searches in the router:

```sh
GOCACHE=/tmp/httpserve-go-build go test -tags purego -run '^$' \
  -bench '^(BenchmarkRouterStdlib|BenchmarkRouterByteSearch|BenchmarkRouterFanout)$' \
  -benchmem -benchtime=100ms -count=10 -cpu=1 > /tmp/router-standard.txt
GOCACHE=/tmp/httpserve-go-build go test -run '^$' \
  -bench '^(BenchmarkRouterStdlib|BenchmarkRouterByteSearch|BenchmarkRouterFanout)$' \
  -benchmem -benchtime=100ms -count=10 -cpu=1 > /tmp/router-neon.txt
benchstat /tmp/router-standard.txt /tmp/router-neon.txt
```

For the original-source comparison, export the baseline commit into a temporary
directory with `git archive`, copy `router_simd_benchmark_test.go` into it, and
build with `go test -c`. Build a second baseline binary after applying the control
patch, and build the current tree as the third binary. Run each binary with
`-test.run '^$' -test.bench '^(BenchmarkRouterStdlib|BenchmarkRouterByteSearch|BenchmarkRouterFanout)$'
-test.benchmem -test.benchtime=100ms -test.count=1 -test.cpu=1` ten times,
alternating before/control/after and after/control/before order. Append each
binary's output to a separate file, then compare with `benchstat`.

For isolated searches, `BenchmarkRouteIndexByte` compares the standard routine
and routing helper across lengths, substring offsets, and first/last/missing
matches. Its timings do not predict the whole router's performance by themselves.
The existing `BenchmarkRouterMatch`, `BenchmarkRouterServeHTTP`, and
`BenchmarkRouterAllocations` suites cover additional route shapes and misses.

## Validation

Passed with the final production implementation:

```sh
GOCACHE=/tmp/httpserve-go-build go test ./...
GOCACHE=/tmp/httpserve-go-build go test -race ./...
GOCACHE=/tmp/httpserve-go-build go test -tags purego ./...
GOCACHE=/tmp/httpserve-go-build go vet ./...
GOCACHE=/tmp/httpserve-go-build go build ./...
GOCACHE=/tmp/httpserve-go-build go test -run '^$' -bench . -benchtime=1x -cpu=1
GOCACHE=/tmp/httpserve-go-build go test -run '^$' \
  -fuzz '^FuzzRouteIndexByte$' -fuzztime=30s -parallel=4
GOCACHE=/tmp/httpserve-go-build go test -run '^$' \
  -fuzz '^FuzzRouterTrie$' -fuzztime=30s -parallel=4
```

The byte-search fuzzer completed 1,977,737 executions; the differential router
fuzzer completed 1,884,961. Deterministic tests cover every match position across
lengths 0–31 and 32 substring offsets, all byte values, repeated matches, and
matches outside the supplied slice. Guard-page tests place strings of lengths
0–31 against inaccessible pages to detect out-of-bounds assembly loads.
The existing tests cover route precedence, fallback, captures, allocations,
and concurrent requests. Additional integration tests exercise captures and
child tables on both sides of the 32-byte cutover, including NUL and non-ASCII
parameter values. All 451 benchmark correctness checks passed.

Test binaries also cross-compiled for `linux/arm64` and `darwin/amd64` with
`CGO_ENABLED=0`. These binaries were not executed; runtime validation was on
`darwin/arm64`. The existing listener tests required local socket access outside
the restricted sandbox. No network-throughput or application benchmark was run.
