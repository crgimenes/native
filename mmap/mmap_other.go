//go:build !unix && !windows

// Fallback for platforms without an mmap backend (e.g. js, plan9). Keeps the
// package building for every GOOS; operations fail with ErrUnsupported.

package mmap

import "os"

func mapFile(f *os.File) (MMap, error) { return nil, ErrUnsupported }

func unmap(m MMap) error { return ErrUnsupported }
