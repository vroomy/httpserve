# Prefix branch versus main and Go's ServeMux

Captured on 2026-10-18 on Apple M3, macOS 26.5.2, Go 1.27.0,
`darwin/arm64`, `GOARM64=v8.0`, `CGO_ENABLED=0`. `GOFLAGS`, `GODEBUG`, and
`GOEXPERIMENT` were empty. Benchstat version:
`golang.org/x/perf v0.0.0-20240707193131-dc66afd55b77`.

- Main: `97491b3991bea36b8bd7d50ec3715bdcb4766bc1`, verified against freshly
  fetched `origin/main`. It already contains the NEON and SSE2 optimizations.
- Branch: `ec85ad99cbb2247d1592d4eaf1233d7251806bb4`, on
  `perf/trie-prefix-fast-paths`, after merging main.

Each binary was built from its complete commit snapshot with `go test -trimpath
-c`. Both snapshots have identical module versions and
[`BenchmarkRouterStdlib`](../../router_stdlib_benchmark_test.go) fixtures. The
only production-code difference is the empty/one-byte prefix specialization in
`routeNode.find`. The branch also includes the new regression tests. The earlier
[main-versus-SIMD report](../router-pr-comparison.md) used an older main revision;
its gains are not attributed to this prefix change.

## Method

[run.py](run.py) runs main and branch serially for ten rounds, alternating their
order. Each binary runs all eight existing HTTPServe/ServeMux cases with
`-test.run '^$' -test.bench '^BenchmarkRouterStdlib$' -test.benchmem
-test.benchtime=1s -test.count=1 -test.cpu=1` and profiling disabled. No builds,
tests, fuzzing, or other agent benchmarks ran alongside timing. External machine
activity and processor scheduling are not controlled.

Each fixture has 40 GET routes, one named parameter, and a successful middle
lookup. Both routers reuse a prepared request and discarding writer, read the
parameter through their native API, and write a 204 status. HTTPServe includes
Context pooling and panic recovery. Registration, request creation, network I/O,
and application work are excluded. These are warmed microbenchmarks, not
end-to-end HTTP throughput measurements. See the
[fixture guide](../README.md#standard-library-comparison) for path and method
semantics that this comparison does not cover.

## This branch versus ServeMux

Medians over ten samples from the branch binary. Speedup is ServeMux time divided
by HTTPServe time, calculated before rounding the displayed timings.

| URL fixture | ServeMux | HTTPServe | Speedup |
| --- | ---: | ---: | ---: |
| Common (19 bytes) | 128.80 ns/op | 35.98 ns/op | 3.58× |
| Short (9 bytes) | 95.50 ns/op | 38.14 ns/op | 2.50× |
| Medium (39 bytes) | 217.50 ns/op | 38.59 ns/op | 5.64× |
| Long (172 bytes) | 646.00 ns/op | 39.10 ns/op | 16.52× |

All four timing differences have `p < 0.001`. HTTPServe uses **0 B/op and
0 allocs/op** in every sample; ServeMux uses **16 B/op and 1 alloc/op**.

## HTTPServe on main versus this branch

Main already includes the SIMD work. These measurements compare HTTPServe with
itself using the same toolchain and fixtures. Speedup is main time divided by
branch time; the last column is the percentage reduction relative to main.

| URL fixture | Main (`97491b3`) | Branch (`ec85ad9`) | Speedup | Less time |
| --- | ---: | ---: | ---: | ---: |
| Common (19 bytes) | 39.23 ns/op | 35.98 ns/op | 1.09× | 8.27% |
| Short (9 bytes) | 41.88 ns/op | 38.14 ns/op | 1.10× | 8.93% |
| Medium (39 bytes) | 42.27 ns/op | 38.59 ns/op | 1.10× | 8.70% |
| Long (172 bytes) | 42.49 ns/op | 39.10 ns/op | 1.09× | 8.00% |

All four HTTPServe timing differences have `p < 0.001`, with no significant
regressions in these fixtures. Both revisions use **0 B/op and 0 allocs/op**.

Two unchanged ServeMux controls also improved significantly between binaries:
Short by **1.26%** (`p = 0.002`) and Medium by **1.11%** (`p = 0.041`). Common
(`p = 0.425`) and Long (`p = 0.342`) did not change significantly. The control
shifts may reflect machine variation or binary layout, so the table reports
observed branch differences without attributing every percentage point to the
prefix comparison. This result is not pooled with the earlier
[isolated prefix measurements](../trie-prefix/README.md), whose controls showed
no significant change. Significance uses benchstat's default 0.05 threshold
without a multiple-comparison correction.

## Validation and saved evidence

All 16 single-iteration preflight cases and **160 timed samples** passed their
handler, parameter, and status checks. Every case has ten samples per revision;
allocation bytes and counts are unchanged. See the [sample audit](validation.txt).
The implementation's prior full,
purego, race, vet, build, and differential fuzz checks remain documented in the
[implementation report](../trie-prefix/README.md#correctness-checks). The merge
added no Go-source or module changes relative to that validated implementation.
This follow-up changes benchmark artifacts and documentation only. Native AMD64
performance remains unmeasured; prior AMD64 correctness execution used Rosetta.

- [Environment, commit IDs, and source/binary hashes](environment.txt).
- Preflight correctness output: [main](main-preflight.txt),
  [branch](branch-preflight.txt).
- Raw samples: [main](samples/main.txt), [branch](samples/branch.txt).
- [Execution order and exact benchmark commands](samples/order.txt).
- Benchstat: [main versus branch](main-comparison.txt),
  [branch versus ServeMux](stdlib-comparison.txt).

## Reproduction

Run from the repository root on native ARM64, after other CPU-heavy work has
finished. Build both snapshots before starting the serial timing capture.

```sh
prefix_work=$(mktemp -d /tmp/httpserve-prefix-pr.XXXXXX)
export GOCACHE=/tmp/httpserve-go-build
export GOOS=darwin GOARCH=arm64 GOARM64=v8.0 CGO_ENABLED=0
export GOFLAGS= GODEBUG= GOEXPERIMENT=
mkdir "$prefix_work/main" "$prefix_work/branch"
git archive 97491b3991bea36b8bd7d50ec3715bdcb4766bc1 |
  tar -x -C "$prefix_work/main"
git archive ec85ad99cbb2247d1592d4eaf1233d7251806bb4 |
  tar -x -C "$prefix_work/branch"
(
  cd "$prefix_work/main"
  go test -trimpath -c -o "$prefix_work/main.test" .
)
(
  cd "$prefix_work/branch"
  go test -trimpath -c -o "$prefix_work/branch.test" .
)
python3 benchmarks/trie-prefix-pr/run.py \
  "$prefix_work/main.test" "$prefix_work/branch.test" "$prefix_work/samples"
benchstat "$prefix_work/samples/main.txt" "$prefix_work/samples/branch.txt"
benchstat -row /URL -col /Router "$prefix_work/samples/branch.txt"
```
