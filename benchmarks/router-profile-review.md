# ARM64 profile after the SIMD changes

Re-profiled the current working tree on 2026-10-17 with Go 1.27.0,
`darwin/arm64`, Apple M3, `GOARM64=v8.0`, and one benchmark CPU. This includes
the [selective SIMD implementation](router-simd.md). This review adds measurements
and an isolated prototype; it does not apply another production optimization.

## Findings

There is still a useful, small optimization to pursue: bypass general string
equality for empty and one-byte trie prefixes. A prototype improved the four
ordinary HTTP fixtures by 8.6–9.4% and deep HTTP fixtures by about 10%, with
unchanged allocations. This is the strongest **validated next step** from this
review. It does not require additional assembly.

The remaining larger opportunities concern traversal and short parameter scans.
They need separate experiments; a large profile share alone does not establish
that the corresponding work can be removed.

## Current CPU costs

Each case ran separately with `-test.benchtime=5s`, CPU and heap sampling, and
`-test.cpu=1`. The profiles include benchmark calibration, so their total
duration can exceed five seconds. All selected warmed HTTP fixtures reported
**0 B/op and 0 allocs/op**. The fixtures reuse requests and run minimal handlers;
they exclude network I/O, request construction, JSON/form processing, and
application work.

| Fixture | Trie traversal, cumulative | Prefix equality, flat | Custom byte search, flat | Context reset, cumulative |
| --- | ---: | ---: | ---: | ---: |
| Common URL, 40 routes | 48.7% | 9.0% | 8.5% | 13.7% |
| Long URL, 40 routes | 57.1% | 16.7% | 10.4% | 9.7% |
| Five-parameter HTTP hit | 77.4% | 6.8% | 27.6% | 6.2% |
| Failed parameter branch, wildcard fallback | 77.1% | 8.3% | 24.6% | 5.1% |
| Rotating 16-child HTTP lookup | 47.6% | 6.8% | 5.9% | 13.3% |
| Direct wildcard lookup, long URL, 1,000 routes | 79.0% | 19.7% | 18.2% | — |

Traversal is `routeNode.find`, equality is `runtime.memequal`, and custom search
is `routeIndexByteNEON`. Equality and search are included in traversal's
cumulative share: **these columns must not be added together**. The custom
routine includes scalar paths as well as NEON. CPU percentages are sampled
attribution, not estimates of achievable speedup. Recursive source listings can
count a sample at several frames; use the function-level cumulative percentages
above rather than summing those listings.

[Saved profile summaries and source attribution](router-profile-cpu.txt) include
the profiled benchmark output. Raw profiles and their binary were retained under
`/tmp/httpserve-profile-review` and `/tmp/httpserve-current-profile.test` locally;
they are temporary artifacts. The owned-parameter `Match` CPU profile was
dominated by `runtime.kevent` and is excluded from CPU hotspot conclusions. Its
allocation profile still identifies `routeEndpoint.prepareParams` as essentially
all sampled allocation objects: one 32-byte allocation per successful
one-parameter owned result. Callers that can reuse results already have
`MatchInto`; warmed HTTP dispatch already reuses its storage.

## Measured prefix prototype

[`routeNode.find`](../routeNode.go) currently calls `strings.HasPrefix` at every
node, including empty root/parameter prefixes and one-byte radix edges. The
prototype switches on prefix length: empty prefixes match directly, one-byte
prefixes check one byte after a length check, and longer prefixes retain
`strings.HasPrefix`. The general equality routine already has short cases, but
this avoids calling it for those nodes.

The [exact prototype patch](router-profile-prefix.patch) was applied only through
a Go source overlay in `/tmp`. Both binaries otherwise use the current source,
including the SIMD changes. Eight paired rounds ran serially, alternating binary
order, with 100 ms per fixture and one CPU. These timing runs had profiling
disabled. No builds, fuzzing, or other benchmarks ran alongside them.

| Fixture | Current median | Prefix prototype | Time reduction |
| --- | ---: | ---: | ---: |
| Common URL HTTP | 39.70 ns | 35.99 ns | 9.37% |
| Short URL HTTP | 41.74 ns | 37.87 ns | 9.27% |
| Medium URL HTTP | 42.15 ns | 38.27 ns | 9.18% |
| Long URL HTTP | 42.55 ns | 38.90 ns | 8.58% |
| Five-parameter HTTP hit | 99.13 ns | 89.32 ns | 9.90% |
| HTTP wildcard fallback | 96.47 ns | 86.41 ns | 10.42% |
| Rotating 16-child HTTP lookup | 30.18 ns | 27.50 ns | 8.88% |
| Direct wildcard lookup, long URL, 1,000 routes | 39.78 ns | 32.39 ns | 18.59% |

All rows shown were significant: `p < 0.001`, except fallback (`p = 0.010`)
and 16-child lookup (`p = 0.007`). All 16 focused cases remained allocation-free.
Static HTTP dispatch and the 4-/64-child cases were noisy and showed no
statistically significant difference; their median changes are not evidence of
a gain or regression. These are measurements on this machine, not guarantees
for every route table or processor.

Saved data: [current](router-profile-current.txt),
[prototype](router-profile-prefix.txt), and
[benchstat comparison](router-profile-prefix-comparison.txt).

A broader screening run covered all **162** existing `RouterMatch` and
`RouterServeHTTP` combinations, with six 30 ms samples per case and alternating
binary order. The geometric mean fell from **25.27 to 23.24 ns/op (8.02%)**.
There were 119 statistically significant improvements and no significant timing
regressions in this capture. Allocation counts and bytes were unchanged. The
short samples make this a regression screen, not a guarantee across workloads;
the longer focused measurements above support the headline improvements.

Matrix data: [current](router-profile-matrix-current.txt),
[prototype](router-profile-matrix-prefix.txt), and
[comparison](router-profile-matrix-comparison.txt). The selector was
`^BenchmarkRouter(Match|ServeHTTP)$`, with `-test.benchtime=30ms -test.cpu=1`.

## Other potential work

1. **Parameter scanning and traversal.** Deep hit/fallback profiles spend
   roughly a quarter of sampled CPU time in the byte-search routine. Source
   attribution puts most of that in parameter delimiter searches. These
   fixtures have one-character parameters with nearby delimiters. Avoiding call
   overhead or specializing such early delimiters is a plausible next experiment,
   but additional branches could hurt longer IDs. Traversal itself also remains
   substantial; iteration through static-only chains could reduce recursive
   calls, provided static/parameter/wildcard fallback and deferred capture writes
   remain correct. Neither change has been measured here.
2. **Context reuse and cleanup.** Reset plus pool Get/Put account for roughly
   24–26% in the ordinary fixtures. There is no per-request allocation left to
   eliminate there. Cleanup deliberately clears the full backing capacity of
   parameters and hooks, including slices shortened by a handler; simply
   clearing fewer elements would violate that guarantee. Smaller fast paths may
   help, but require measurement. Panic recovery and reference cleanup remain
   part of the measured behavior.
3. **Application work.** The current suite cannot identify a bottleneck in
   networking, serialization, middleware, or business logic. A representative
   application profile is needed to rank those against another few nanoseconds
   of routing. These microbenchmarks do not establish that the whole server has
   no further optimization opportunities.

## Reproduce profiling

```sh
mkdir -p /tmp/httpserve-profile-review
GOCACHE=/tmp/httpserve-go-build go test -c \
  -o /tmp/httpserve-current-profile.test .
/tmp/httpserve-current-profile.test -test.run '^$' \
  -test.bench '^BenchmarkRouterStdlib$/^URL=Common$/^Router=Trie$' \
  -test.benchtime=5s -test.cpu=1 -test.benchmem \
  -test.cpuprofile=/tmp/httpserve-profile-review/common.cpu \
  -test.memprofile=/tmp/httpserve-profile-review/common.mem
GOCACHE=/tmp/httpserve-go-build go tool pprof -top \
  /tmp/httpserve-current-profile.test /tmp/httpserve-profile-review/common.cpu
```

The other selectors are `BenchmarkRouterStdlib/URL=Long/Router=Trie`,
`BenchmarkRouterAllocations/DeepHit/ServeHTTP`,
`BenchmarkRouterAllocations/Fallback/ServeHTTP`,
`BenchmarkRouterFanout/Children=16`,
`BenchmarkRouterMatch/Long/Wildcard/Routes=1000/LastHit`, and
`BenchmarkRouterAllocations/Param/Match`. Anchor each path component as above
to avoid matching similarly named subbenchmarks.

For prototype comparisons, build the current source and a separate copy with
the saved patch, then alternate the binaries using the existing benchmark
fixtures. The prototype's selected router, Context, parser, allocation, SIMD
boundary, and differential seed tests passed. Ten seconds of differential
router fuzzing passed **636,076 executions**. This is candidate validation;
cross-platform and full integration validation remain necessary before adopting
the prototype as a production change.
