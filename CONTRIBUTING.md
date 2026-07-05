# Contributing to native

Thanks for your interest. native is a collection of small, focused Go packages
that bind OS APIs **without cgo**. Two rules define it; a change that fights
them will be declined however well written.

## 1. No cgo. Ever.

The point of the project. Backends reach the OS through
[purego](https://github.com/ebitengine/purego), `syscall`, `dlopen` /
`LoadLibrary`, the Objective-C runtime, and COM; there is no C compiler in the
loop, which is what keeps `CGO_ENABLED=0` builds, six-target cross-compilation
from one machine, and a `go install` that needs no toolchain.

Concretely: no `import "C"`, no dependency that needs cgo, nothing bundled
(no `.so`/`.dylib`/`.dll` shipped; we call what the OS already has). purego is
the only dependency and that list does not grow.

A feature that can only be done with cgo does not belong here.

## 2. YAGNI: the least that works

Each package does one job with the smallest API that covers it. Before adding
anything, walk the ladder: does it need to exist? is it already here? does the
stdlib do it? does a platform API cover it? can it be a few lines? Only then
write the minimum.

No speculative options, no interface with one implementation, no "we might need
it later". When you deliberately take a shortcut with a known ceiling, mark it
in the code (`// yagni: <ceiling>, <upgrade trigger>`); several packages already
do this.

Never minimize away validation, error handling, security, bounded execution, or
the one test that proves the logic.

## All three platforms, or honest about it

A package should work on macOS, Windows, and Linux. Where a platform genuinely
cannot do something cheaply and safely (fragmented desktop APIs, D-Bus-only
routes; Linux is usually the hard case), it returns `ErrUnsupported` and the
README says so. Honest-unsupported is fine; pretending is not.

## The shape of a package

Every package follows the same layout; see the README "Conventions" section and
use `clipboard/` as the template:

- Public API in a tag-free `foo.go` (doc comments, exported types, sentinel
  errors) delegating to per-platform files, plus a `foo_other.go` so every
  `GOOS` builds.
- No native types in signatures; inputs and outputs are plain Go values.
- A package `README.md` (API table, per-platform status, caveats) and a
  runnable example under `examples/<pkg>/`.
- Tests that run the real backend where the CI runner has one and skip
  honestly where it doesn't.

The ABI discipline in the README ("the part that bites") is required reading
before touching FFI: no Go pointer held by native code; struct-by-value is
arch-specific and breaks at runtime, not compile time; thread affinity is real;
Windows has no `Dlopen`.

## Public API stability

This is a public library. Exported API does not get removed; if something turns
out wrong, document it and supersede it. Signature changes are still possible
pre-1.0, but treat them as expensive.

## Scope

Standalone, window-independent bindings live here. Anything that needs an
application window or run loop (dialogs, menus) belongs in
[glaze](https://github.com/crgimenes/glaze) instead. If a platform surface is
too unstable to bind honestly, the answer is `ErrUnsupported`, not a flaky
backend.

## Code style

`gofmt`, US English everywhere. No inline `if` init; assign on its own line,
then `if`. No `else` after a terminal branch; return early. Comments explain
why, not what.

## Before you open a PR

```sh
go fix ./...
gofmt -l .                    # must print nothing
go vet ./...
for os in darwin linux windows; do GOOS=$os golangci-lint run ./...; done
gosec ./...                   # audited findings use // #nosec Gxxx, never bare //nosec
go test -timeout 90s -count 1 ./...
make cross                    # compile-check all six GOOS/GOARCH
```

CI runs the suite for real on macOS, Windows, and Linux runners; a change is not
done until all jobs are green. Cross-compilation does not catch ABI bugs; the
runners do.

## Proposing a change

The default branch is `master`. Doc fixes and small patches can go straight to a
PR. For a new package or a new backend, open an issue first with the per-OS
approach; agreeing on scope and the `ErrUnsupported` boundaries up front saves
everyone time.
