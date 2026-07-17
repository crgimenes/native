// Package nocapture keeps a window's content out of screenshots, screen
// recordings and screen sharing, cgo-free — captures show the window as a
// black rectangle while the user keeps seeing it normally, the way DRM-locked
// video players appear in a shared screen.
//
//	err := nocapture.Protect(w.Window()) // e.g. a glaze WebView handle
//
// Platform truth, stated plainly:
//
//   - Windows: SetWindowDisplayAffinity(WDA_MONITOR). Supported and reliable.
//
//   - macOS: ErrUnsupported. The old -[NSWindow setSharingType:
//     NSWindowSharingNone] stopped working in macOS 15.4 (ScreenCaptureKit
//     ignores it), Apple documents the constant as legacy, and DTS has stated
//     there is no public API for preventing screen capture. Worse, on macOS 26
//     the legacy value can stop the window from rendering at all, so this
//     package refuses to set it.
//
//   - Linux: ErrUnsupported. Neither X11 nor Wayland exposes a portable
//     equivalent.
//
// This is a privacy courtesy enforced by the OS compositor, not content DRM:
// it does nothing against a camera pointed at the display.
package nocapture

import (
	"errors"
	"unsafe"
)

// ErrUnsupported is returned on a platform with no working backend: Linux has
// no compositor API for it, and macOS removed the one it had (see the package
// comment).
var ErrUnsupported = errors.New("nocapture: not supported on this platform")

// Protect marks the window's content as capture-protected: captures and
// screen sharing show it blacked out. The handle is the toolkit's native
// window pointer — an HWND on Windows, exactly what glaze's WebView.Window()
// returns. Call it from the UI thread, after the window exists.
func Protect(window unsafe.Pointer) error {
	if window == nil {
		return errors.New("nocapture: nil window handle")
	}
	return protect(window)
}
