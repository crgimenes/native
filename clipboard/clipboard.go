// Package clipboard provides cgo-free access to the system clipboard.
//
// The API is intentionally tiny: read and write UTF-8 text, and images as PNG
// bytes. Each platform binds the clipboard the OS already provides — NSPasteboard on macOS, the Win32
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

// ReadImage returns the clipboard's image as PNG bytes, converting from the
// platform's other image formats when needed. No image yields nil and a nil
// error. Implemented on macOS; elsewhere ErrUnsupported.
func ReadImage() ([]byte, error) { return readImage() }

// WriteImage replaces the clipboard's content with a PNG image.
func WriteImage(png []byte) error { return writeImage(png) }
