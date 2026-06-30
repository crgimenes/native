// Package filedialog shows the operating system's native open and save file
// panels, cgo-free.
//
// Threading: the panels are platform UI (AppKit, GTK, Win32) and must be invoked
// on the program's main thread. This package does not impose a threading model —
// the caller is responsible for already being on the main thread. For example,
// an Ebitengine app wraps the call in ebiten.RunOnMainThread, and a webview host
// uses its own UI-thread dispatch.
//
// Each function returns the chosen path, or "" when the user cancels (or on a
// platform without an implementation yet).
package filedialog

// Options configures a file panel. The zero value is valid: a default panel
// rooted at the platform's default directory with no type filtering.
type Options struct {
	// Title is the prompt shown prominently above the file list.
	Title string

	// Directory is the initial directory, as a filesystem path. Empty uses the
	// platform default (usually the last-used directory).
	Directory string

	// Filename is the suggested file name. Used by Save and ignored by Open.
	Filename string

	// Extensions restricts selectable files to these extensions, given without
	// the leading dot (e.g. {"afoil", "dat"}). Empty, or any "*"/"" entry,
	// allows all files.
	Extensions []string
}
