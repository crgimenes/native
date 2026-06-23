// Package clipboard provides cgo-free access to the system text clipboard.
//
// The API is intentionally tiny: read and write UTF-8 text. Each platform binds
// the clipboard the OS already provides — NSPasteboard on macOS, the Win32
// clipboard on Windows, and X11/Wayland on Linux — with no cgo and no bundled
// native libraries.
//
//	old, _ := clipboard.ReadText()
//	clipboard.WriteText("hello")
//
// Text is exchanged as UTF-8 Go strings; the backend handles any conversion the
// platform needs (UTF-16 on Windows, NSString on macOS).
package clipboard

import "errors"

// ErrUnsupported is returned by operations on a platform that has no clipboard
// backend wired up yet.
var ErrUnsupported = errors.New("clipboard: not supported on this platform")

// ReadText returns the clipboard's current text content. An empty clipboard (or
// one holding only non-text data) yields an empty string and a nil error.
func ReadText() (string, error) { return readText() }

// WriteText replaces the clipboard's content with the given UTF-8 text.
func WriteText(s string) error { return writeText(s) }
