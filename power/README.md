# power

Keep the system awake, cgo-free. `PreventSleep` asks the OS not to idle-sleep
until you release the returned token — the keep-awake you want around a long
copy, a build, or media playback.

```go
import "github.com/crgimenes/native/power"

tok, err := power.PreventSleep("ripping a disc")
if err != nil {
	// errors.Is(err, power.ErrUnsupported) where there is no backend
	log.Fatal(err)
}
defer tok.Release()

// ... do the long-running work; the machine won't idle-sleep ...
```

## API

| Func | Description |
| --- | --- |
| `PreventSleep(reason string) (*Token, error)` | Inhibit idle system sleep until the token is released. `reason` is a short label some platforms surface (it becomes the macOS assertion name) and others ignore. |
| `(*Token) Release() error` | End the inhibition. Idempotent — extra or concurrent calls are safe no-ops. |
| `ErrUnsupported` | Sentinel returned by a platform with no backend (Linux). |

The token is the whole API surface. No native handles cross the boundary.

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS | IOKit `IOPMAssertionCreateWithName` (`PreventUserIdleSystemSleep`) | ✅ |
| Windows | `SetThreadExecutionState(ES_CONTINUOUS \| ES_SYSTEM_REQUIRED)` | ✅ builds + CI |
| Linux | — | `ErrUnsupported` |

Check the unsupported case with `errors.Is(err, power.ErrUnsupported)`.

**Why Linux is unsupported.** There is no portable, cgo-free way to inhibit
sleep on Linux. The supported route is the systemd `org.freedesktop.login1`
`Inhibit` method, which hands back a file descriptor over **D-Bus** — and a
cgo-free D-Bus client means either a new dependency (this module avoids them) or
a hand-rolled wire-protocol client with `SCM_RIGHTS` fd passing, which is fragile
across distributions. Rather than ship something flaky, Linux returns a clear
`ErrUnsupported`. If a real need arrives, the login1 inhibitor (or shelling out
to `systemd-inhibit`) is the path.

## Notes

- **What is inhibited:** idle *system* sleep. The display is still allowed to
  sleep. (`PreventUserIdleSystemSleep` on macOS; `ES_SYSTEM_REQUIRED` without
  `ES_DISPLAY_REQUIRED` on Windows.)
- **Effect vs. binding.** A test can confirm the call succeeds and the token
  round-trips, but not that the machine stayed awake. Verify the live effect by
  hand: `pmset -g assertions` on macOS, `powercfg /requests` on Windows, both of
  which list the active assertion while the example runs.
- **Battery / power-source state is not here.** It is a separate,
  side-effect-free concern with no consumer yet; it can be added later behind its
  own call.

## Example

A runnable demo (inhibit, hold a few seconds, release) lives in
[`examples/power`](../examples/power):

```bash
go run ./examples/power
```

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `power.go`, per-platform `power_darwin.go` and `power_windows.go`, and a
`power_other.go` (`!darwin && !windows`) that returns `ErrUnsupported` so every
`GOOS` builds.
