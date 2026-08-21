# filedialog

Native open/save/choose-directory panels, cgo-free.

```go
path := filedialog.Open(filedialog.Options{
    Title:      "Open scene",
    Extensions: []string{"afoil"},
})
if path != "" {
    // user picked a file
}

out := filedialog.Save(filedialog.Options{
    Title:    "Save scene",
    Filename: "untitled.afoil",
})

dir := filedialog.PickDirectory(filedialog.Options{
    Title: "Choose migrations directory",
})
```

Each call returns the chosen path, or `""` when the user cancels, when the
platform has no implementation, or when the panel cannot be shown (for example a
Linux process with no display). There is no error to inspect; a desktop app
treats all of those the same way — nothing was picked.

## Threading

The panels are platform UI and **must be called on the main thread**. This
package deliberately does not impose a threading model; the caller arranges to be
on the main thread. For an Ebitengine app:

```go
var path string
ebiten.RunOnMainThread(func() {
    path = filedialog.Open(filedialog.Options{Extensions: []string{"afoil"}})
})
```

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS | `NSOpenPanel` / `NSSavePanel` via the objc runtime (`PickDirectory` is `NSOpenPanel` in directory mode) | ✅ runs locally |
| Windows | Common Item Dialog (`IFileOpenDialog` / `IFileSaveDialog`, COM via purego) | ✅ builds + CI; not yet exercised on real hardware |
| Linux | `GtkFileChooserNative` (GTK3 or GTK4, chosen at runtime) | ✅ validated on real hardware (Debian 13, GTK3, xvfb) |

### Linux notes

- **GTK stack selection.** Loading GTK3 and GTK4 into one process corrupts the
  GObject type system, so the package first probes (`RTLD_NOLOAD`, loads
  nothing) for a GTK the process **already** has — a glaze/GTK host app — and
  joins it. Only when neither is mapped does it load one fresh: GTK3 first, then
  GTK4.
- **Modal loop.** `gtk_native_dialog_run` was removed in GTK4, so the modal is
  driven manually (`set_modal` + `show` + the `response` signal + main-loop
  iteration) — the same mechanism `run` used internally, valid on both GTK3 and
  GTK4.
- **Headless.** With no display `gtk_init_check` fails and every panel returns
  `""` instead of crashing; the package's GTK smoke test skips itself the same
  way (and runs for real under xvfb in CI).
