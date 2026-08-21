//go:build !darwin && !windows && !linux

// Stub for platforms without a native panel implementation (e.g. *BSD, js).
// Every panel reports cancellation so callers degrade gracefully, keeping the
// module building for every GOOS.

package filedialog

func open(_ Options) string { return "" }

func save(_ Options) string { return "" }

func pickDirectory(_ Options) string { return "" }
