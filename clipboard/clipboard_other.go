//go:build !darwin && !windows && !linux

// Fallback for platforms without a clipboard backend (e.g. *BSD, Plan 9, js).
// Keeps the module building for every GOOS; operations fail with ErrUnsupported.

package clipboard

func readText() (string, error) { return "", ErrUnsupported }

func writeText(s string) error { return ErrUnsupported }

// yagni: images only on macOS until a consumer needs them here; Windows would
// take CF_DIB, which other apps read, besides a registered PNG format.
func readImage() ([]byte, error) { return nil, ErrUnsupported }

func writeImage(_ []byte) error { return ErrUnsupported }
