# singleinstance

Enforce a single running instance of an application and forward the arguments of
later launches to the one already running, cgo-free. macOS/Linux use an `flock`'d
lock file plus a Unix-domain socket; Windows uses a named pipe — no C toolchain,
no bundled native libraries.

```go
import "github.com/crgimenes/native/singleinstance"

inst, err := singleinstance.Acquire("com.example.app", singleinstance.Options{
    OnMessage: func(args []string) { /* a later launch passed these */ },
})
if errors.Is(err, singleinstance.ErrAlreadyRunning) {
    _ = singleinstance.Send("com.example.app", os.Args[1:]) // forward and exit
    return
}
defer inst.Release()
```

## How it works

`Acquire` either becomes the **primary** (returns an `*Instance` you hold for the
app's lifetime) or reports `ErrAlreadyRunning`. A secondary launch then calls
`Send` to hand its arguments to the primary, which receives them on
`Options.OnMessage`, and exits. This is the standard "focus/open-in-existing-
window" pattern: the second click forwards the file/URL to the app already up.

The lock is released automatically when the primary process dies (the `flock` is
dropped on exit; the Windows pipe instance is destroyed), so there is no stale
lock to clean up.

## API

| Func | Description |
| --- | --- |
| `Acquire(id string, opts Options) (*Instance, error)` | Become the single instance for `id`, or get `ErrAlreadyRunning`. |
| `Send(id string, args []string) error` | Forward `args` to the running instance (after `ErrAlreadyRunning`). |
| `(*Instance) Release() error` | Drop the lock and stop listening. Idempotent. |
| `Options.OnMessage func([]string)` | Called on the primary with a later launch's args. |
| `ErrAlreadyRunning`, `ErrUnsupported` | Sentinels. |

`id` is any stable application identifier (e.g. a reverse-DNS string); it is
hashed into a bounded, safe lock/socket/pipe name. Arguments cross as a
`[]string` (JSON on the wire); no native types cross the API boundary.

## Platforms

| OS | Lock | Hand-off | Status |
| --- | --- | --- | --- |
| macOS | `flock` lock file | Unix-domain socket | ✅ |
| Linux | `flock` lock file | Unix-domain socket | ✅ |
| Windows | named pipe (`FILE_FLAG_FIRST_PIPE_INSTANCE`) | same named pipe | ✅ builds + CI |

macOS and Linux share one Unix backend (`//go:build unix`, standard library
only). Check the unsupported case with `errors.Is(err, singleinstance.ErrUnsupported)`.

## Example

A runnable demo (run it twice to see the hand-off) lives in
[`examples/singleinstance`](../examples/singleinstance):

```bash
go run ./examples/singleinstance            # primary
go run ./examples/singleinstance hello      # secondary, forwards "hello"
```

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `singleinstance.go` with `ErrUnsupported`, a shared `_unix.go` backend
for macOS/Linux, `_windows.go`, and `_other.go` so every `GOOS` builds.
