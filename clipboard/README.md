# clipboard

Read and write the system **text** clipboard, cgo-free. Each platform binds the
clipboard the OS already provides — `NSPasteboard` on macOS, the Win32 clipboard
on Windows, X11/Wayland on Linux — with no C toolchain and no bundled native
libraries.

```go
import "github.com/crgimenes/native/clipboard"

old, _ := clipboard.ReadText()
clipboard.WriteText("hello")
```

Text is exchanged as UTF-8 Go strings; the backend handles whatever the platform
needs underneath (UTF-16 on Windows, `NSString` on macOS).

## API

| Func | Description |
| --- | --- |
| `ReadText() (string, error)` | Current clipboard text. Empty (or non-text) clipboard yields `"", nil`. |
| `WriteText(s string) error` | Replace the clipboard content with `s`. |
| `ErrUnsupported` | Sentinel returned by a platform with no backend wired up yet. |

No native types cross the API boundary — just `string` and `error`.

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS | `NSPasteboard` via the Objective-C runtime | ✅ tested on hardware |
| Windows | Win32 clipboard (`OpenClipboard`/`SetClipboardData`, `CF_UNICODETEXT`) | ✅ builds + CI |
| Linux | X11 selection (`libX11` via purego) | ⬜ returns `ErrUnsupported` for now |

Check for the unsupported case with `errors.Is(err, clipboard.ErrUnsupported)`.

### Linux note (once it lands)

The X11 backend will own the `CLIPBOARD` selection directly via `libX11` (no GTK,
to avoid the GTK3/GTK4 load clash). X11 clipboard ownership is held by a live
process: written text survives only while the program runs, unless a clipboard
manager grabs it. That's an X11 property, not a bug — the docs will say so when
the backend ships.

## Example

A runnable demo (write, read back, restore the previous content) lives in
[`examples/clipboard`](../examples/clipboard):

```bash
go run ./examples/clipboard
```

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `clipboard.go`, per-platform `clipboard_{darwin,windows,linux}.go`, and
`clipboard_other.go` returning `ErrUnsupported` so every `GOOS` builds.
