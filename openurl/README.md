# openurl

Open URLs in the user's default handler and reveal files in the file manager,
cgo-free. Each platform uses what the OS already ships — `NSWorkspace` on macOS,
`ShellExecuteW` on Windows, `xdg-open` on Linux — with no C toolchain and no
bundled native libraries.

```go
import "github.com/crgimenes/native/openurl"

openurl.Open("https://example.com")
openurl.Reveal("/path/to/file")
```

## Security

`Open` only accepts `http`, `https`, `mailto` and `file` URLs. Any other scheme
— or a bare string with no scheme — is rejected with `ErrScheme`, so a hostile
value can't launch an arbitrary protocol handler (`javascript:`, a custom app
scheme, etc.). The string is never passed through a shell; on Linux it is a
single `argv` entry to `xdg-open`, so it can't be word-split or expanded.

## API

| Func | Description |
| --- | --- |
| `Open(rawurl string) error` | Open `rawurl` with the default handler. Rejects schemes outside the allow-list with `ErrScheme`. |
| `Reveal(path string) error` | Show `path` in the file manager. The path must exist. |
| `ErrScheme` | Sentinel for a URL whose scheme is not allowed. |
| `ErrUnsupported` | Sentinel returned by a platform with no backend wired up yet. |

No native types cross the API boundary — just `string` and `error`.

## Platforms

| OS | Open | Reveal | Status |
| --- | --- | --- | --- |
| macOS | `-[NSWorkspace openURL:]` | `activateFileViewerSelectingURLs:` (selects the file) | ✅ |
| Windows | `ShellExecuteW("open", …)` | `explorer /select,` (selects the file) | ✅ builds + CI |
| Linux | `xdg-open` | `xdg-open` on the containing folder | ✅ |

Check for the unsupported case with `errors.Is(err, openurl.ErrUnsupported)`.

### Reveal on Linux

`Reveal` highlights the file on macOS and Windows. On Linux it opens the
*containing folder* instead: selecting a specific file is file-manager specific
(`nautilus --select`, `dolphin --select`, …) and not portable, so `openurl` opens
the directory rather than guessing the file manager. The proper portable path is
the XDG desktop portal (`org.freedesktop.portal.OpenURI`); it can be added later
without changing this API.

## Example

A runnable demo (open a page, reveal a temp file) lives in
[`examples/openurl`](../examples/openurl):

```bash
go run ./examples/openurl
```

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `openurl.go`, per-platform `openurl_{darwin,windows,linux}.go`, and
`openurl_other.go` returning `ErrUnsupported` so every `GOOS` builds.
