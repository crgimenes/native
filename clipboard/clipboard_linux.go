// Linux clipboard backend.
//
// NOT YET IMPLEMENTED — returns ErrUnsupported. Linux is the hard case and is
// the next task for this package; it needs verification on real hardware/CI, so
// it is left explicit rather than shipped blind.
//
// Design notes for the implementation:
//
//   - X11 (libX11 via purego.Dlopen "libX11.so.6"): the X clipboard is not a
//     buffer you write to — it is an ownership protocol. ReadText opens a
//     display, creates an unmapped window, calls XConvertSelection(CLIPBOARD,
//     UTF8_STRING) and reads the data back from the SelectionNotify event via
//     XGetWindowProperty. WriteText must XSetSelectionOwner(CLIPBOARD) and then
//     keep serving SelectionRequest events from a background goroutine for as
//     long as the data should persist — when the process exits the content is
//     gone unless a clipboard manager grabbed it. That persistent-owner loop is
//     the real work here and why a buffer-style API is a leaky fit on X11.
//   - Wayland (wl_data_device): harder still — needs a live Wayland connection
//     and a focused surface. Likely best routed through the XDG desktop portal
//     (org.freedesktop.portal.Clipboard) over D-Bus where available.
//
// Until then the package builds on Linux and fails cleanly.

package clipboard

func readText() (string, error) { return "", ErrUnsupported }

func writeText(s string) error { return ErrUnsupported }
