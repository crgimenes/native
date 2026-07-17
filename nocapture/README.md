# nocapture

Keeps a window's content out of screenshots, screen recordings and screen
sharing: captures show the window as a black rectangle while the user keeps
seeing it normally — the way DRM-locked video players appear in a shared
screen.

```go
err := nocapture.Protect(w.Window()) // e.g. a glaze WebView handle
```

| Platform | Backend | Effect |
|----------|---------|--------|
| Windows | `SetWindowDisplayAffinity(WDA_MONITOR)` | window blacked out in captures |
| macOS | — | `ErrUnsupported` (see below) |
| Linux | — | `ErrUnsupported`: no portable compositor API |

**Why no macOS backend.** The old `-[NSWindow setSharingType:
NSWindowSharingNone]` stopped working in macOS 15.4 — ScreenCaptureKit
ignores it — Apple now documents the constant as legacy, and Apple DTS has
stated there is no public API for preventing screen capture. Worse: on
macOS 26 setting the legacy value can stop the window from rendering at all.
A backend that either does nothing or breaks the window would be worse than
an honest `ErrUnsupported`.

Call it from the UI thread once the window exists. The handle is the
toolkit's native window pointer (HWND), exactly what glaze's
`WebView.Window()` returns.

This is a privacy courtesy enforced by the OS compositor, not content DRM: it
does nothing against a camera pointed at the display.
