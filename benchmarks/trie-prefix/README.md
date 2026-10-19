# Implemented trie prefix comparisons

Measured on 2026-10-18 on Apple M3, macOS 26.5.2, Go 1.27.0,
`darwin/arm64`, `GOARM64=v8.0`, `CGO_ENABLED=0`, with `GOFLAGS`, `GODEBUG`,
and `GOEXPERIMENT` empty. Benchstat version:
`golang.org/x/perf v0.0.0-20240707193131-dc66afd55b77`.

## Change and baseline

[`routeNode.find`](../../routeNode.go) now accepts an empty prefix without a
comparison, checks a one-byte prefix directly after guarding an empty URL, and
retains `strings.HasPrefix` for longer prefixes. Traversal, slash boundaries,
fallback order, and deferred parameter capture are unchanged. This is portable
Go; it also applies in purego builds and adds no assembly.

The baseline is clean commit `e1b90bfc65c57878ebd8604066c3e9f3f8ce258d`,
including the completed ARM64 NEON and AMD64 SSE2 changes. Its source snapshot,
binary, and benchmark correctness output were captured before editing. Work is
on `perf/trie-prefix-fast-paths`. The candidate benchmark binary uses that same
snapshot with **only `routeNode.go` replaced** in a separate source copy, so
even test and benchmark sources are identical. Both builds use `-trimpath` and
the same toolchain and environment. Full correctness checks use the working tree,
including the added regression tests.

This implements the [historical prototype](../router-profile-review.md), whose
roughly 9% ordinary HTTP improvement and 8% matrix reduction were preliminary.
Those measurements are preserved separately and are not pooled with this run.

## Method

[run.py](run.py) runs the binaries serially for ten paired rounds per suite,
reversing baseline/candidate order every other round. Each process uses
`-test.run '^$' -test.count=1 -test.cpu=1 -test.benchmem` and no profiling.

- Focused: 14 cases, one second per case. Four 40-route ordinary HTTP fixtures
  each include HTTPServe and the unchanged `http.ServeMux` control. Deep hit,
  deep miss, and failed-parameter/wildcard-fallback cases each measure
  `MatchInto` and `ServeHTTP`.
- Matrix: all 162 `BenchmarkRouterMatch` and `BenchmarkRouterServeHTTP` cases,
  100 ms per case. These combine short/medium/long URLs, static/parameter/wildcard
  routes, 1/100/1,000 routes, and first hit/last hit/miss. This remains a shorter
  regression screen; the one-second focused samples support the HTTP conclusions.

No builds, tests, fuzzing, or other agent benchmarks ran alongside timing.
External machine activity, thermal state, and processor scheduling were not
controlled. These warmed microbenchmarks exclude network I/O, request creation,
and application work. They do not measure end-to-end HTTP throughput. The
[fixture guide](../README.md#standard-library-comparison) describes the differing
HTTPServe and ServeMux semantics.

## Focused native ARM64 results

Medians across ten samples per version; percentages are time reductions relative
to the SIMD baseline. Every HTTPServe row below is significant at `p < 0.001`.

| Fixture | SIMD baseline | Prefix implementation | Less time |
| --- | ---: | ---: | ---: |
| Common URL HTTP | 38.94 ns | 35.95 ns | 7.69% |
| Short URL HTTP | 41.68 ns | 37.95 ns | 8.95% |
| Medium URL HTTP | 42.21 ns | 38.49 ns | 8.79% |
| Long URL HTTP | 42.47 ns | 38.99 ns | 8.18% |
| Five-parameter HTTP hit | 101.60 ns | 91.53 ns | 9.91% |
| Deep HTTP miss | 95.89 ns | 88.07 ns | 8.15% |
| HTTP wildcard fallback | 98.34 ns | 88.17 ns | 10.34% |
| Five-parameter `MatchInto` hit | 91.99 ns | 82.51 ns | 10.31% |
| Deep `MatchInto` miss | 77.57 ns | 70.25 ns | 9.44% |
| `MatchInto` wildcard fallback | 82.23 ns | 74.53 ns | 9.36% |

All ten HTTPServe cases retain **0 B/op and 0 allocs/op** in every sample.
The unchanged ServeMux controls show no significant differences between binaries
(`p = 0.138–0.897`) and retain **16 B/op and 1 alloc/op**. No focused case
regressed significantly. All 280 samples passed their benchmark correctness
checks. See the [focused benchstat comparison](focused-comparison.txt).

The rerun also supplies a fresh comparison against ServeMux using the candidate's
samples. Ratios below are ratios of medians; all differences have `p < 0.001`.
These successful GET fixtures use each router's native parameter API and the
same minimal handler, with the limitations described above.

| URL fixture | ServeMux | HTTPServe | HTTPServe speedup |
| --- | ---: | ---: | ---: |
| Common (19 bytes) | 127.50 ns | 35.95 ns | 3.55× |
| Short (9 bytes) | 96.16 ns | 37.95 ns | 2.53× |
| Medium (39 bytes) | 220.25 ns | 38.49 ns | 5.72× |
| Long (172 bytes) | 645.15 ns | 38.99 ns | 16.55× |

See the [ServeMux benchstat comparison](stdlib-comparison.txt). README benchmark
numbers were updated only because this corresponding comparison was rerun.

## Broader native ARM64 matrix

The geometric mean of the 162 per-case median times fell from **24.55 to
22.86 ns/op (6.89% less time)**. Benchstat reports **153 significant improvements,
nine cases without a significant change, and no significant regressions**.
All **3,240** samples passed their benchmark correctness checks. The matrix's
100 ms samples are longer than the historical prototype's 30 ms screen, and
there are ten samples per version instead of six. The fresh 6.89% reduction is
separate from the historical 8.02%; the old result is not the implementation's
headline measurement.

Some short matrix cases had wide confidence intervals. The only increased median
was medium-URL static HTTP dispatch, 100 routes, last hit: 42.67 to 43.05 ns/op
(about 0.9% higher, `p = 0.912`). This is not a statistically established regression.

Bytes and allocations are unchanged in every case: the 18 successful owned
parameter `Match` cases use 32 B/op and 1 alloc/op, and the remaining 144 cases
use 0 B/op and 0 allocs/op. All warmed HTTP cases are allocation-free.

See the [complete matrix comparison](matrix-comparison.txt) and
[sample-count/allocation audit](summary.txt). Significance counts use benchstat's
default `p < 0.05` threshold without a multiple-comparison correction. The
geometric mean weights every fixture equally; it is not an estimate of a
particular application's request mix. No significant timing or allocation
regression was found in either suite, but native AMD64 performance and
end-to-end HTTP workloads remain unmeasured.

## Correctness checks

- `gofmt` on all three changed Go files; `git diff --check`.
- Full `go test -count=1 ./...` and `go test -tags purego -count=1 ./...`
  on native ARM64 and AMD64 under Rosetta 2.
- Native ARM64 `CGO_ENABLED=1 go test -race -count=1 ./...`.
- `go vet ./...` and `go build ./...` for ARM64 and AMD64 targets.
- `FuzzRouterTrie`, 30 seconds with four workers on each architecture:
  **1,911,630 executions on ARM64**, **1,167,803 under Rosetta**; no failures.
- All **253** router benchmark correctness cases passed with `-benchtime=1x`
  on the baseline and candidate ARM64 binaries and the AMD64 candidate.
  These single-iteration outputs are correctness evidence, not timing results.
- The new node-prefix and expanded trie-matching tests also pass against the
  original baseline implementation via an overlay. They cover empty input,
  one-byte mismatch, unchanged capture storage, static fallback, and slash
  boundaries. Existing tests retain the broader precedence and rollback coverage.

Existing HTTP integration tests bind port 8080 and needed localhost socket access
outside the sandbox. The initial sandbox restriction and successful reruns are
recorded in [checks.txt](checks.txt). The referenced `docs/development.md` is
absent from this checkout. No unrelated documentation or runtime repairs were
made. AMD64 execution here is **translated**, not native Intel/AMD performance
evidence. No AMD64 timings are claimed.

## Reproduction

From this change's repository root, prepare isolated benchmark binaries. The
candidate substitutes only the production source so newly added tests cannot
affect the comparison's binary layout. Keep the source unchanged during builds.

```sh
prefix_repo=$PWD
prefix_work=$(mktemp -d /tmp/httpserve-prefix.XXXXXX)
export GOCACHE=/tmp/httpserve-go-build
export GOOS=darwin GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=0
export GOFLAGS= GOEXPERIMENT= GODEBUG=
mkdir "$prefix_work/source"
git archive e1b90bfc65c57878ebd8604066c3e9f3f8ce258d |
  tar -x -C "$prefix_work/source"
(
  cd "$prefix_work/source"
  go test -trimpath -c -o "$prefix_work/baseline.test" .
)
cp -R "$prefix_work/source" "$prefix_work/candidate"
cp "$prefix_repo/routeNode.go" "$prefix_work/candidate/routeNode.go"
# This should report only routeNode.go.
diff -rq "$prefix_work/source" "$prefix_work/candidate" || test "$?" -eq 1
(
  cd "$prefix_work/candidate"
  go test -trimpath -c -o "$prefix_work/candidate.test" .
)
# Run after all builds and correctness checks finish; use a new output directory.
python3 benchmarks/trie-prefix/run.py \
  "$prefix_work/baseline.test" "$prefix_work/candidate.test" "$prefix_work/samples"
benchstat "$prefix_work/samples/focused-baseline.txt" \
  "$prefix_work/samples/focused-candidate.txt"
benchstat "$prefix_work/samples/matrix-baseline.txt" \
  "$prefix_work/samples/matrix-candidate.txt"
benchstat -filter '.name:RouterStdlib' -row /URL -col /Router \
  "$prefix_work/samples/focused-candidate.txt"
```

Full validation commands and outputs are in [checks.txt](checks.txt); the AMD64
commands set `GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0`. To repeat differential
fuzzing, run `go test -run '^$' -fuzz '^FuzzRouterTrie$' -fuzztime=30s -parallel=4`
once per architecture. Use `CGO_ENABLED=1` for the ARM64 race run. Tests that
open localhost listeners need socket permission.

## Saved evidence

- [Discarded setup run](discarded-overlay-samples/README.md): an unapplied
  temporary-path overlay produced identical binaries. This interrupted run is
  excluded from all results; the corrected runner rejects identical binaries.
- [Environment, source hashes, binary hashes, and fixture hashes](environment.txt).
- [Baseline](baseline-preflight.txt), [candidate](candidate-preflight.txt), and
  [Rosetta](rosetta-preflight.txt) benchmark correctness output.
- [Execution order and exact commands](samples/order.txt).
- Focused samples: [baseline](samples/focused-baseline.txt),
  [candidate](samples/focused-candidate.txt).
- Matrix samples: [baseline](samples/matrix-baseline.txt),
  [candidate](samples/matrix-candidate.txt).
- Benchstat: [focused](focused-comparison.txt), [matrix](matrix-comparison.txt),
  and [candidate versus ServeMux](stdlib-comparison.txt).
