# AMD64 short routing searches

## Implementation

[`routeIndexByte_amd64.go`](../routeIndexByte_amd64.go) extends the inlined
empty-string and first-byte checks to AMD64. Remaining short searches call
[`routeIndexByte_amd64.s`](../routeIndexByte_amd64.s): scalar loads for 1–3 bytes,
SSE2 comparisons for 4–31 bytes. It requires only `GOAMD64=v1` instructions;
no AVX, SSSE3, BMI instructions, or runtime CPU detection are needed.

For 4–7 bytes, the first and last four bytes are packed into the low eight bytes
of a vector. The unused upper lanes are masked out, including for a NUL target.
For 8–15 bytes, the first and last eight bytes fill a vector. For 16–31 bytes,
the routine checks the first 16 bytes and, if necessary, the last 16. Overlaps
are intentional; the first match is returned. Every load stays within the
provided string, including unaligned strings and strings next to guard pages.

The matcher retains `strings.IndexByte` for searches of 32 bytes or more.
`-tags purego` disables this package's assembly and its inlined shortcuts; other
architectures retain the same standard-library fallback. Go's standard library
can still use SIMD in those builds. The compile-time dispatch constant was
renamed from `useRouteIndexByteNEON` to `useRouteIndexByteSIMD`; ARM64's search
algorithm and routing semantics are unchanged. The separate prefix-specialization
prototype remains unapplied.

## Validation environment

Validated on 2026-10-18 with Go 1.27.0 on Apple M3. AMD64 binaries were built
with `GOOS=darwin GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0` and executed under
**Rosetta 2**. This exercises AMD64 code and its ABI, but does not establish
performance on native Intel or AMD hardware. The headline README benchmark
remains the measured native ARM64 comparison with `http.ServeMux`.

The existing byte-search tests cover every match position for lengths 0–31,
32 substring offsets, repeated matches, all target byte values, and targets
outside the supplied slice. Guard-page coverage now runs on AMD64 as well as
ARM64, on Darwin and Linux. Router tests exercise parameter boundaries across
the 32-byte cutover and child counts of 15, 16, 31, 32, 33, and 64.

Checks completed:

- Full `go test -count=1 ./...` suites: native ARM64 and AMD64 under Rosetta.
- Full `go test -tags purego -count=1 ./...` suites on both architectures.
- Native ARM64 `go test -race -count=1 ./...`.
- `go vet ./...` and `go build ./...` on ARM64 and AMD64.
- AMD64 `FuzzRouteIndexByte`, 20 seconds: **848,358 executions**.
- AMD64 `FuzzRouterTrie`, 20 seconds: **529,608 executions**.
- All **451** benchmark correctness cases with `-benchtime=1x` on each
  architecture. These single-iteration runs are correctness checks, not timings.
- Test binaries cross-compiled for Linux/AMD64, Windows/AMD64, and Linux/RISC-V64
  (generic fallback); these binaries were not executed.
- Compiler diagnostics confirm the AMD64 wrapper inlines at both matcher call
  sites. Formatting and documentation links were checked.

An additional Linux/386 cross-compilation check failed in the existing Sonic
dependency, whose `internal/utils/skip.go` intentionally rejects 32-bit integers.
No dependency or 32-bit support changes are included here. Existing HTTP tests
required localhost socket access outside the sandbox.

## Translated timing experiment

These results describe **Rosetta on Apple M3, not native AMD64 hardware**.
Six rounds ran serially, rotating binary order, with 100 ms per fixture and
`-cpu=1`. No builds, tests, or fuzzing ran alongside the timing capture. The
25 fixtures cover parameter lengths, rotating sibling lookups, and the existing
standard-library comparison, with correctness checks before and after timing.

- **Before:** commit `4ba3dab`, whose AMD64 path used `strings.IndexByte`.
- **Fast paths:** the same runtime behavior with empty-string/first-byte checks
  written directly in the matcher and the standard library handling other
  searches. This uses the existing [control patch](router-simd-fastpaths.patch)
  applied to `routeNode.go` from `f8c7f8f`, then substituted with a Go source
  overlay. Writing those checks directly avoids an extra non-inlined Go wrapper.
- **After:** the current inlined wrapper and bounded SSE2 routine.

All binaries used Go 1.27.0, `GOAMD64=v1`, and `CGO_ENABLED=0`. Benchstat version:
`golang.org/x/perf v0.0.0-20240707193131-dc66afd55b77`.

| HTTP fixture | Before | Fast paths | SSE2 implementation |
| --- | ---: | ---: | ---: |
| Common URL | 69.97 ns | 58.63 ns | 54.52 ns |
| Short URL | 84.12 ns | 63.14 ns | 62.95 ns |
| Medium URL | 84.60 ns | 64.05 ns | 63.29 ns |
| Long URL | 85.53 ns | 64.82 ns | 63.81 ns |

Each HTTPServe fixture improved versus the original path (`p = 0.002`), and all
retained zero allocations. Versus the equally inlined standard-library control,
Common improved 7.02% (`p = 0.002`); the other fixture differences were not
significant. This supports retaining the short-search approach for this
translated workload, but is not evidence of native Intel/AMD speedups.

Unchanged ServeMux controls varied significantly between the before and after
binaries: Common +4.19%, Medium +7.23%, Long +9.26%. The control's Medium trie
fixture also had a noisy outlier. Translation, binary layout, and machine
variation limit conclusions from small timing differences; no native hardware
recommendation is derived from these measurements. The earlier ARM64 report
likewise separates the effect of inlining from custom SIMD.

Saved samples: [before](router-amd64-rosetta-before.txt),
[fast paths](router-amd64-rosetta-fastpaths.txt), and
[after](router-amd64-rosetta-after.txt). Statistical comparisons:
[before/after](router-amd64-rosetta-comparison.txt) and
[fast paths/SSE2](router-amd64-rosetta-sse2-comparison.txt).

A broader Rosetta screening run covered all **162** `BenchmarkRouterMatch` and
`BenchmarkRouterServeHTTP` cases with six 30 ms samples per case, alternating
before/after binary order. The geometric mean changed from **42.99 to 34.36 ns/op
(20.09% less time)**, with 160 significant improvements and no significant timing
regressions. Allocation counts and bytes were unchanged. These shorter samples
are a translated regression screen, not a substitute for native AMD64 profiling.
Saved matrix: [before](router-amd64-rosetta-matrix-before.txt),
[after](router-amd64-rosetta-matrix-after.txt), and
[comparison](router-amd64-rosetta-matrix-comparison.txt).

## Reproduction

On a native AMD64 machine, run the existing fixtures with and without the custom
short-search path to measure its effect on that processor:

```sh
GOCACHE=/tmp/httpserve-go-build GOAMD64=v1 go test -tags purego -run '^$' \
  -bench '^(BenchmarkRouterStdlib|BenchmarkRouterByteSearch|BenchmarkRouterFanout)$' \
  -benchmem -benchtime=1s -count=10 -cpu=1 > /tmp/router-amd64-standard.txt
GOCACHE=/tmp/httpserve-go-build GOAMD64=v1 go test -run '^$' \
  -bench '^(BenchmarkRouterStdlib|BenchmarkRouterByteSearch|BenchmarkRouterFanout)$' \
  -benchmem -benchtime=1s -count=10 -cpu=1 > /tmp/router-amd64-simd.txt
benchstat /tmp/router-amd64-standard.txt /tmp/router-amd64-simd.txt
```

The purego comparison includes both the inlined shortcuts and custom search
routine, rather than isolating the effect of vector instructions. Repeat the
broader `BenchmarkRouterMatch` and `BenchmarkRouterServeHTTP` matrix for the
route shapes relevant to an application.

To exercise AMD64 correctness on an Apple Silicon Mac with Rosetta installed:

```sh
GOCACHE=/tmp/httpserve-go-build GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0 \
  go test -count=1 ./...
GOCACHE=/tmp/httpserve-go-build GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0 \
  go test -run '^$' -fuzz '^FuzzRouteIndexByte$' -fuzztime=20s -parallel=4
GOCACHE=/tmp/httpserve-go-build GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0 \
  go test -run '^$' -fuzz '^FuzzRouterTrie$' -fuzztime=20s -parallel=4
```
