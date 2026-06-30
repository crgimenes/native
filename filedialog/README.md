# filedialog

Native open/save file panels, cgo-free.

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
```

Each call returns the chosen path, or `""` when the user cancels.

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

- macOS: `NSOpenPanel` / `NSSavePanel` via purego's Objective-C runtime.
- Linux, Windows: not implemented yet (the calls return `""`); GTK and Win32
  backends can be ported in.
