# Testing and coverage

Run checks from the repository root with the Go version required by `go.mod`:

```sh
go test ./...
go test -tags purego ./...
go test -race ./...
go vet ./...
go build ./...
```

Listener tests require local socket access. Existing tests in `engine_test.go`
use port 8080; keep it free. New listener tests use OS-assigned ports and wait
for startup under the engine's ownership lock, then close and join the server.
Certificate tests generate temporary self-signed certificates. Automatic TLS
tests check local configuration and startup without contacting an ACME service.
The directory-permission test skips on systems that do not enforce read permissions.

If the default Go build cache is read-only, set `GOCACHE` to a writable directory,
for example `GOCACHE=/tmp/apex-go-cache`.

## Measuring library coverage

```sh
go test -coverprofile=/tmp/apex-coverage.out . ./form
go tool cover -func=/tmp/apex-coverage.out
go tool cover -html=/tmp/apex-coverage.out -o /tmp/apex-coverage.html
```

The explicit package list measures the two library packages. `go test ./...`
also checks the example executables; including those in a coverage profile adds
their untested `main` functions to the total.

Measured on 2026-10-19 with Go 1.27.2 on darwin/arm64:

| Package | Before | After |
| --- | ---: | ---: |
| `github.com/vroomy/apex` | 65.2% | 99.7% |
| `github.com/vroomy/apex/form` | 84.8% | 100.0% |

The combined library statement coverage is 99.7%. Go's coverage percentage
measures executed statements, not every branch, input, or behavior. Assembly is
not instrumented. The portable `purego` build reports 99.3% for Apex and 100%
for `form`: it also counts two SIMD dispatch statements that its build-time
constant disables. Native ARM64 and portable `purego` builds were checked;
these results do not establish coverage on other architectures or verify live
ACME certificate issuance.

Tests cover request binding and body closure, response bodies and metadata,
write and encoding errors, automatic redirects, handler completion, hook order,
engine callbacks, listener ownership and shutdown, TLS discovery and validation,
host policies, HTTPS upgrades, and form decoding failures. Existing tests cover
trie matching, parameter ownership, middleware isolation, allocations, and SIMD
search boundaries.

## Remaining uncovered statements

Two statements remain unreachable through the current implementation:

- [NewUpgrader](../upgrader.go#L12) handles failure to register its constant
  `GET /*` route on a newly constructed engine. That registration cannot fail
  under the current router implementation (two statements).

This path remains in the profile. No statements are excluded or made injectable
to raise the coverage percentage. The former `Router.Handle` parser-error branch
was removed when `getParts` stopped returning its unused error result.

## Behavior findings for separate fixes

These issues were confirmed while reviewing and exercising the code; their
runtime behavior remains unchanged:

- [JSONValue.Errors](../jsonValue.go#L61) stores `[]error`. Encoding an ordinary
  `errors.New("failure")` with `NewJSONResponse(400, ...)` emits
  `{"errors":[{}]}`, losing the message. Decoding that response through
  `UnmarshalJSONValue` fails to unmarshal into `error` instead of reconstructing
  the original error. The existing error test only checks for a non-nil result,
  so it does not prove error round-tripping works.
- [JSONResponse.WriteTo](../jsonResponse.go#L57) and
  [JSONPResponse.WriteTo](../jsonPResponse.go#L52) return a byte count of zero
  even when writing a nonempty body. Tests validate their bodies and errors
  without asserting this incorrect count as the desired contract.
- [Decoder.processEquals](../form/decoder.go#L82) treats every literal `=` as
  a separator: `token=a=b` decodes to `ab` instead of `a=b`. Also,
  [processAmpersand](../form/decoder.go#L86) returns before resetting state for
  an empty pair: `=&name=Ada` leaves `name` unset. Percent-encoded `=` values
  decode correctly and are covered.
