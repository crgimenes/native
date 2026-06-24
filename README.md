# native

[![test](https://github.com/crgimenes/native/actions/workflows/ci.yml/badge.svg)](https://github.com/crgimenes/native/actions/workflows/ci.yml)

Small, focused Go packages that bind native OS and hardware APIs **without cgo**.

Each package fills one obvious gap in the standard library — clipboard, desktop
notifications, file-system watching, serial ports, mmap, and so on — with a
clean Go API and a per-platform backend underneath. No C toolchain, no bundled
native libraries: backends call what the OS already ships, via
[`purego`](https://github.com/ebitengine/purego), `syscall`,
`dlopen`/`LoadLibrary`, the Objective-C runtime, COM, and D-Bus.

These are siblings of [glaze](https://github.com/crgimenes/glaze) (a pure-Go
WebView binding) and grew out of the same idea: you can port surprisingly
complex native integration to Go and keep everything cgo gives up — trivial
cross-compilation (`CGO_ENABLED=0`), reproducible builds, and a `go install`
that just works for whoever clones the repo.

## Philosophy

- **Small over sprawling.** One package, one job. No "parallel stdlib", no
  god-package. Each directory here is independently useful and independently
  testable, but they compose into a real desktop app when used together.
- **Idiomatic Go at the seams.** The public API is plain Go — `string`,
  `[]byte`, `error`, small option structs. Native handles (`objc.ID`, COM
  vtables, `HWND`, file descriptors) never leak across the package boundary, so
  the Win32/Cocoa/portal backend can change without touching callers.
- **Zero cgo, always.** `CGO_ENABLED=0` builds and cross-compiles for every
  supported target.
- **Honest about platforms.** Where a platform genuinely can't do something
  cheaply (Linux clipboard ownership, Wayland global hotkeys), the package
  returns a clear sentinel error and the README says so, rather than pretending.

## Non-goals

Deliberately *not* attempted here — these are maintenance black holes or already
well-served elsewhere:

- A complete GUI toolkit (use glaze + HTML, or a real toolkit).
- Full audio engines / DSP, Vulkan/Metal/WebGPU, generic USB stacks.
- "Perfect" Wayland coverage. macOS and Windows are tractable; Linux desktop is
  fragmented and gets best-effort, portal-first support.

## Packages

The ordered build plan (value × ease, per-package approach, and the cgo-free
gotchas carried over from glaze) lives in [TODO.md](TODO.md). Status here:
✅ done · 🚧 in progress · ⬜ planned · 🟦 lives in
[glaze](https://github.com/crgimenes/glaze) instead (window/app-coupled).

| Package          | What                                             | macOS | Windows | Linux | Status |
| ---------------- | ------------------------------------------------ | ----- | ------- | ----- | ------ |
| [`clipboard`](clipboard/) | Read/write text clipboard                   | NSPasteboard | Win32 clipboard | X11/Wayland | 🚧 |
| `notify`         | Desktop notifications                            | UNUserNotification | WinRT toast | D-Bus | ⬜ |
| `tray`           | System tray / status-bar icon + menu             | NSStatusItem | Shell_NotifyIcon | StatusNotifierItem | ⬜ |
| `keychain`       | Credential / secret storage                      | Keychain | Credential Manager / DPAPI | Secret Service | ⬜ |
| `fswatch`        | File-system change notifications                 | FSEvents/kqueue | ReadDirectoryChangesW | inotify | ⬜ |
| `serial`         | Serial port I/O                                  | termios | DCB/CreateFile | termios | ⬜ |
| `mmap`           | Memory-mapped files / shared memory              | mmap | MapViewOfFile | mmap | ⬜ |
| `singleinstance` | Single-instance lock + arg hand-off to running   | flock/socket | named mutex/pipe | flock/socket | ⬜ |
| [`openurl`](openurl/) | Open URL in browser, reveal file in file manager | NSWorkspace | ShellExecuteW | xdg-open | ✅ |
| `power`          | Inhibit sleep, battery/power state               | IOKit assertions | SetThreadExecutionState | systemd/UPower | ⬜ |

### Lives in glaze, not here

Features that need the application's window/run loop belong in the desktop
framework, tracked in [glaze's `TODO.md`](https://github.com/crgimenes/glaze/blob/main/TODO.md):

- **File dialogs** (open/save/choose-directory) — modal, parented to the window.
  glaze already drives `NSOpenPanel` for `<input type=file>`.
- **Native application menus** — the macOS menu bar, About/Preferences/Quit,
  window menus on Windows/Linux.

The standalone packages here (`clipboard`, `notify`, `openurl`, `tray`, …) are
imported *by* glaze; they don't depend on it.

## Conventions

Every package follows the same shape so they're predictable to use and to write:

- **One package per directory**, named for the job (`clipboard`, `serial`).
- **Platform split by build tags / filenames**: `foo_darwin.go`,
  `foo_windows.go`, `foo_linux.go`, and a `foo_other.go`
  (`//go:build !darwin && !windows && !linux`) that returns `ErrUnsupported`,
  so the module always builds for every `GOOS`.
- **Public API in a tag-free file** (`foo.go`): doc comments, exported types,
  and sentinel errors live there and delegate to unexported per-platform funcs.
- **Sentinel errors**, e.g. `var ErrUnsupported = errors.New("clipboard: not supported on this platform")`.
- **No native types in signatures.** Inputs/outputs are Go values.
- **A package `README.md`** (API table, per-platform status, caveats) **and a
  runnable example** in `examples/<pkg>/` — same module, no extra deps. See
  [`clipboard`](clipboard/) for the shape.
- **`golangci-lint` clean on every `GOOS`.** The `unsafe.Pointer(uintptr)` that
  `go vet`'s `unsafeptr` flags (common with purego — e.g. a `GlobalLock` pointer)
  is spelled through a `ptr(u uintptr) unsafe.Pointer` reinterpret helper, which
  is ABI-identical and silences the false positive.

### ABI discipline (the part that bites)

cgo's pointer rules still apply conceptually even without cgo. The scars, learned
porting glaze's three backends:

- **No Go pointer is ever held by native code.** The GC can move Go memory. Pass
  an integer id, keep a `map[id]*T` on the Go side, resolve it in the callback.
- **Anything handed to native code must stay alive and unmovable** — keep it in a
  package-level slice/map (callback trampolines, COM vtables).
- **Struct-by-value is architecture-specific.** A 16-byte struct goes by hidden
  reference on Win64 amd64 but packed into registers on arm64 (AAPCS64) — needs
  `*_amd64.go` / `*_arm64.go`. **This does not show up in cross-compilation; it
  compiles clean and breaks at runtime.**
- **`purego.NewCallback` on Windows** is limited: ≤9 uintptr args, no float /
  struct-by-value, single uintptr return. Design inbound callbacks within that.
- **Thread affinity.** GUI/COM APIs are pinned to a thread (Cocoa → real main
  thread; GTK → the `gtk_init` thread; COM apartment + pump → its thread). Use
  `runtime.LockOSThread` and route cross-thread calls through a dispatch.

## Testing reality

ABI bugs do not appear in cross-compilation — you have to run on the target.
CI (GitHub Actions) builds, vets and tests on macOS, Windows and Linux runners,
runs `make cross` to compile-check every `GOOS/GOARCH`, and runs golangci-lint
across each `GOOS`. Where a package has a backend on the runner its tests run for
real (`clipboard`'s round trip on macOS and Windows); where it doesn't, they skip
(`clipboard` on Linux, for now) instead of failing. A file that says
`// UNTESTED on this host` means exactly that: it compiles, but no human or CI on
that OS has confirmed it yet. Hardware is the final word — even CI runners each
have one clean, single-stack environment and miss real-desktop combinations.

## License

MIT — see [LICENSE](LICENSE).
