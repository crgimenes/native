//go:build !darwin && !windows && !linux

// Fallback for platforms without a clipboard backend (e.g. *BSD, Plan 9, js).
// Keeps the module building for every GOOS; operations fail with ErrUnsupported.

package clipboard

func readText() (string, error) { return "", ErrUnsupported }

func writeText(s string) error { return ErrUnsupported }
