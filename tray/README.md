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
| `SetItems(items []Item) error` | Replace the whole menu while the tray runs. Safe from any goroutine. Returns `ErrNotRunning` when idle. |
| `Config` | `Title`, `Tooltip`, `Icon []byte` (PNG), `Items []Item`, `OnReady func()`. |
| `Item` | `Title`, `Disabled`, `Separator`, `OnClick func()`. |
| `ErrUnsupported`, `ErrAlreadyRunning`, `ErrNotRunning` | Sentinels. |

No native handles cross the boundary.

## Threading

`Run` owns the process's UI event loop, so it must be called from the main
goroutine, locked to the main OS thread (`runtime.LockOSThread()` first thing in
`main`). `Item.OnClick` runs on that UI thread — keep it short or hand work to
another goroutine. `Stop` is the exception: it is safe to call from anywhere (a
menu item's `OnClick` typically just calls `tray.Stop`).

Only one tray runs per process; a second `Run` returns `ErrAlreadyRunning`.

## Starting work when the tray is up (`OnReady`)

`Run` blocks, so the code that needs a live tray has to come from somewhere else.
`OnReady` is that somewhere: it fires once, on the UI thread, after the icon is on
screen and the event loop is turning — no `time.AfterFunc` guess that is either a
race (too short) or a visible lag (too long).

```go
tray.Run(tray.Config{
	Items:   menu("starting"),
	OnReady: func() { go poll() }, // the tray exists by now
})
```

Like `OnClick`, it runs on the UI thread: keep it short, and hand real work to a
goroutine.

## Changing the menu at runtime (`SetItems`)

`Config.Items` is read once by `Run`. To show changing status, flip `Pause` to
`Resume`, or grow a recent-files list, call `SetItems` with the menu you want now
— replacing the whole menu is how items are added, retitled, greyed out, and
removed, so there are no indices to keep in sync.

```go
err := tray.SetItems([]tray.Item{
	{Title: "Status: " + state, Disabled: true},
	{Separator: true},
	{Title: "Quit", OnClick: tray.Stop},
})
```

It is safe from any goroutine: the call stages the items and the rebuild happens
on the UI thread, so it may not have landed yet when `SetItems` returns. With no
tray running it reports `ErrNotRunning` rather than dropping the update.

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS | `NSStatusItem` + `NSMenu` (AppKit via the objc runtime) | ✅ runs locally |
| Windows | `Shell_NotifyIconW` + a hidden window + `TrackPopupMenu` | ✅ builds + CI |
| Linux | — | `ErrUnsupported` |

### Icon support (`Config.Icon`, a PNG)

| OS | Behavior |
| --- | --- |
| macOS | ✅ Renders the PNG, scaled into the menu bar. With no icon it shows `Title`, or a bullet, so the item is always clickable. |
| Windows | ⚠️ **`Config.Icon` is ignored.** The tray shows the application's default icon. Honoring a custom PNG needs a GDI+ PNG→`HICON` conversion — a known follow-up. The tray is always visible regardless. |
| Linux | n/a — `ErrUnsupported` (no backend). |

So a caller can always pass `Config.Icon` and it "shows when possible": a real
icon on macOS, the default app icon on Windows, nothing on Linux.

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

The example also exercises both runtime hooks: it logs from `OnReady` and then
retitles a status item once a second through `SetItems`.

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `tray.go`, per-platform `tray_darwin.go` / `tray_windows.go`, and a
`tray_other.go` (`!darwin && !windows`) that returns `ErrUnsupported` so every
`GOOS` builds.
