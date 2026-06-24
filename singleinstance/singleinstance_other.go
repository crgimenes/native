//go:build !unix && !windows

// Fallback for platforms without a singleinstance backend (e.g. js, plan9).
// Keeps the module building for every GOOS; operations fail with ErrUnsupported.

package singleinstance

func acquire(id string, opts Options) (*Instance, error) { return nil, ErrUnsupported }

func send(id string, args []string) error { return ErrUnsupported }
