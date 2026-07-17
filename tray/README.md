# tray

A system-tray / menu-bar icon with a menu, cgo-free. Build a `Config`, hand it to
`Run`, and `Run` drives the OS event loop until `Stop`.

```go
import (
	"runtime"

	"github.com/crgimenes/native/tray"
)

func main() {
	runtime.LockOSThread() // Run owns the process's UI event loop

	err := tray.Run(tray.Config{
		Title:   "myapp",
		Tooltip: "myapp is running",
		Items: []tray.Item{
			{Title: "Open", OnClick: openUI},
			{Separator: true},
			{Title: "Quit", OnClick: tray.Stop},
		},
	})
	if err != nil {
		// errors.Is(err, tray.ErrUnsupported) where there is no backend (Linux)
		log.Fatal(err)
	}
}
```

## API

| Symbol | Description |
| --- | --- |
| `Run(cfg Config) error` | Show the tray and drive the OS event loop until `Stop`. **Blocks**; call from the main goroutine (locked to the main OS thread). Returns `ErrUnsupported` / `ErrAlreadyRunning`. |
| `Stop()` | Hide the tray and make `Run` return. Safe from any goroutine; no-op when idle. |
| `Config` | `Title`, `Tooltip`, `Icon []byte` (PNG), `Items []Item`. |
| `Item` | `Title`, `Disabled`, `Separator`, `OnClick func()`. |
| `ErrUnsupported`, `ErrAlreadyRunning` | Sentinels. |

No native handles cross the boundary.

## Threading

`Run` owns the process's UI event loop, so it must be called from the main
goroutine, locked to the main OS thread (`runtime.LockOSThread()` first thing in
`main`). `Item.OnClick` runs on that UI thread — keep it short or hand work to
another goroutine. `Stop` is the exception: it is safe to call from anywhere (a
menu item's `OnClick` typically just calls `tray.Stop`).

Only one tray runs per process; a second `Run` returns `ErrAlreadyRunning`.

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS | `NSStatusItem` + `NSMenu` (AppKit via the objc runtime) | ✅ runs locally |
| Windows | `Shell_NotifyIconW` + a hidden window + `TrackPopupMenu` | ✅ builds + CI |
| Linux | — | `ErrUnsupported` |

**Icon.** macOS renders `Config.Icon` (a PNG) scaled into the menu bar; with no
icon it shows `Title` (or a bullet, so the item is always clickable). Windows
currently uses the application's default icon and ignores `Config.Icon` — a
custom Windows icon needs a GDI+ PNG→`HICON` conversion and is a follow-up; the
tray is always visible regardless.

**Why Linux is unsupported.** A Linux tray is a StatusNotifierItem plus a
`com.canonical.dbusmenu` export over **D-Bus** — a dependency this module avoids —
and it is fragmented across desktops (GNOME needs a shell extension to show it at
all). Rather than ship something flaky, Linux returns a clear `ErrUnsupported`.

## Effect vs. binding

A unit test can confirm the unsupported path and that `Stop` is a safe no-op, but
not that an icon appeared and a menu item fired — that needs a display and the
main thread. [`examples/tray`](../examples/tray) is the manual vehicle:

```bash
go run ./examples/tray
```

Set `TRAY_AUTOCLOSE=1` to have it stop itself after a couple of seconds (a
non-interactive smoke test that the icon comes up and the loop tears down).

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `tray.go`, per-platform `tray_darwin.go` / `tray_windows.go`, and a
`tray_other.go` (`!darwin && !windows`) that returns `ErrUnsupported` so every
`GOOS` builds.
