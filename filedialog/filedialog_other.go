// Stub for platforms without a native panel implementation yet (Linux and
// Windows). Open and Save report cancellation so callers degrade gracefully. The
// GTK and Win32 backends can be ported here later (glaze has working versions).

//go:build !darwin

package filedialog

// Open is the no-op stub; see the darwin build for the real implementation.
func Open(_ Options) string { return "" }

// Save is the no-op stub; see the darwin build for the real implementation.
func Save(_ Options) string { return "" }
