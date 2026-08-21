// Package filedialog shows the operating system's native open, save, and
// choose-directory panels, cgo-free.
//
// Threading: the panels are platform UI (AppKit, Win32/COM, GTK) and must be
// invoked on the program's main thread. This package does not impose a
// threading model — the caller is responsible for already being on the main
// thread. For example, an Ebitengine app wraps the call in
// ebiten.RunOnMainThread, and a webview host uses its own UI-thread dispatch.
//
// Each function returns the chosen path, or "" when the user cancels, when the
// platform has no implementation, or when the panel cannot be shown (for
// example a Linux process with no display).
package filedialog

import "strings"

// Options configures a file panel. The zero value is valid: a default panel
// rooted at the platform's default directory with no type filtering.
type Options struct {
	// Title is the prompt shown prominently above the file list.
	Title string

	// Directory is the initial directory, as a filesystem path. Empty uses the
	// platform default (usually the last-used directory).
	Directory string

	// Filename is the suggested file name. Used only by Save.
	Filename string

	// Extensions restricts selectable files to these extensions, given without
	// the leading dot (e.g. {"afoil", "dat"}). Empty, or any "*"/"" entry,
	// allows all files. Ignored by PickDirectory.
	Extensions []string
}

// Open shows a modal open-file panel and returns the chosen path ("" if
// cancelled).
func Open(opts Options) string { return open(opts) }

// Save shows a modal save-file panel and returns the chosen path ("" if
// cancelled). Nothing is written; the panel only picks the path.
func Save(opts Options) string { return save(opts) }

// PickDirectory shows a modal choose-directory panel and returns the chosen
// path ("" if cancelled).
func PickDirectory(opts Options) string { return pickDirectory(opts) }

// cleanExtensions normalizes Options.Extensions for the backends: leading dots
// are stripped, and a nil result means "no restriction" (empty input, or any
// ""/"*" wildcard entry).
func cleanExtensions(exts []string) []string {
	clean := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.TrimPrefix(e, ".")
		if e == "" || e == "*" {
			return nil
		}
		clean = append(clean, e)
	}
	if len(clean) == 0 {
		return nil
	}
	return clean
}
