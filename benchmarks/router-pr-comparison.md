# Main versus the SIMD branch

Captured on 2026-10-18 on Apple M3, Go 1.27.0, `darwin/arm64`,
`GOARM64=v8.0`, with `GODEBUG` and `GOEXPERIMENT` unset.

- **Main:** `f8c7f8fbfed9505a8ad3434c27041e1ff6da77eb`.
- **Branch:** `e5e2b2ed7a5c5430ba9b7f8ffe58809827a14859`.

Both versions used identical `go.mod`, `go.sum`, and
[`BenchmarkRouterStdlib`](../router_stdlib_benchmark_test.go) source. Each fixture
has 40 GET routes, one named parameter, and a successful middle lookup. The
handler reads the parameter and writes a 204 status. Context pooling and panic
recovery are included; request construction, network I/O, and application work
are excluded.

The two binaries ran serially for ten rounds, alternating main/branch order.
Each round ran all eight HTTPServe/ServeMux cases for one second with `-cpu=1`.
No builds, tests, or other benchmarks ran alongside the capture. All 160 samples
passed their handler, parameter, and status checks.

## HTTPServe results

Median times; reductions are calculated relative to main:

| URL fixture | Main | Branch | Less time |
| --- | ---: | ---: | ---: |
| Common (19 bytes) | 47.26 ns/op | 38.98 ns/op | 17.52% |
| Short (9 bytes) | 57.99 ns/op | 41.75 ns/op | 28.00% |
| Medium (39 bytes) | 57.71 ns/op | 42.16 ns/op | 26.95% |
| Long (172 bytes) | 57.41 ns/op | 42.40 ns/op | 26.15% |

All four timing differences have `p < 0.001`, with ten samples per version.
Both versions used **0 B/op and 0 allocs/op** in every HTTPServe sample. The
unchanged ServeMux controls showed no statistically significant timing changes
(`p = 0.183–0.590`) and retained 16 B/op and one allocation per operation.

These results measure the combined inlining and SIMD changes in selected warmed
HTTP fixtures. The [ARM64 report](router-simd.md) separates the effects of those
changes and records the earlier 162-case screening matrix, including two small
direct `Match` regressions. Native Intel/AMD performance remains unmeasured;
the [AMD64 report](router-simd-amd64.md) labels Rosetta validation separately.

Saved data: [main](router-pr-main.txt), [branch](router-pr-branch.txt), and
[statistical comparison](router-pr-comparison.txt). Benchstat version:
`golang.org/x/perf v0.0.0-20240707193131-dc66afd55b77`.

## Reproduction

Export the two commits to separate temporary directories and build each with
`GOCACHE=/tmp/httpserve-go-build go test -c`. Run the resulting binaries once per
round for ten rounds, reversing their order every other round, with these flags:

```sh
-test.run '^$' -test.bench '^BenchmarkRouterStdlib$' \
  -test.benchmem -test.benchtime=1s -test.cpu=1
```

Append each binary's output to its own file, then compare:

```sh
benchstat benchmarks/router-pr-main.txt benchmarks/router-pr-branch.txt \
  > benchmarks/router-pr-comparison.txt
```
