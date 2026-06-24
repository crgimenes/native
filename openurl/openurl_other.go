//go:build !darwin && !windows && !linux

// Fallback for platforms without an openurl backend (e.g. *BSD, Plan 9, js).
// Keeps the module building for every GOOS; operations fail with ErrUnsupported.

package openurl

func openURL(rawurl string) error { return ErrUnsupported }

func revealFile(absPath string) error { return ErrUnsupported }
