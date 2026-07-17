// Package tray puts an icon with a menu in the system tray / menu bar, cgo-free.
//
// Build a Config, hand it to Run, and Run blocks driving the OS event loop until
// Stop is called. Each backend binds what the OS already ships — NSStatusItem on
// macOS, Shell_NotifyIcon on Windows — with no cgo and no bundled libraries.
//
//	err := tray.Run(tray.Config{
//		Title: "myapp",
//		Items: []tray.Item{
//			{Title: "Open", OnClick: openUI},
//			{Separator: true},
//			{Title: "Quit", OnClick: tray.Stop},
//		},
//	})
//
// Threading: Run owns the process's UI event loop, so it must be called from the
// main goroutine, locked to the main OS thread:
//
//	func main() {
//		runtime.LockOSThread()
//		tray.Run(cfg)
//	}
//
// A menu item's OnClick runs on that UI thread; keep it short or hand work to
// another goroutine. Stop, by contrast, is safe to call from any goroutine.
//
// Backends: macOS and Windows are implemented. Linux is not — a StatusNotifierItem
// tray means a D-Bus dependency this module avoids and it is fragmented across
// desktops, so Run returns ErrUnsupported there.
package tray

import "errors"

// ErrUnsupported is returned by Run on a platform with no tray backend.
var ErrUnsupported = errors.New("tray: not supported on this platform")

// ErrAlreadyRunning is returned by Run when a tray is already active in this
// process; only one tray may run at a time.
var ErrAlreadyRunning = errors.New("tray: already running")

// Config describes the tray icon and its menu. It is read once by Run.
type Config struct {
	// Title is a short text label. macOS shows it in the menu bar (next to the
	// icon, or alone when Icon is empty). Windows ignores it.
	Title string

	// Tooltip is shown on hover.
	Tooltip string

	// Icon is a PNG image. macOS renders it in the menu bar (scaled to fit).
	// Windows currently uses the application's default icon and ignores this
	// field (custom Windows icons are a follow-up); it is never nil-checked into
	// an invisible tray.
	Icon []byte

	// Items are the menu entries, top to bottom.
	Items []Item
}

// Item is one entry in the tray menu.
type Item struct {
	// Title is the menu text. Ignored when Separator is true.
	Title string

	// Disabled greys the item out and suppresses OnClick.
	Disabled bool

	// Separator makes this a divider line instead of a clickable item; all
	// other fields are ignored.
	Separator bool

	// OnClick is called on the UI thread when the item is chosen.
	OnClick func()
}

// Run shows the tray and drives the OS event loop until Stop is called. It
// blocks and must be called from the main goroutine (see the package doc on
// threading). It returns ErrUnsupported on platforms with no backend and
// ErrAlreadyRunning if a tray is already active.
func Run(cfg Config) error { return run(cfg) }

// Stop hides the tray and makes Run return. It is safe to call from any
// goroutine and is a no-op when no tray is running.
func Stop() { stop() }
