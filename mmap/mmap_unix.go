//go:build unix

// Unix backend (macOS, Linux, *BSD): syscall.Mmap / syscall.Munmap, straight
// from the standard library. No purego, no extra dependency.

package mmap

import (
	"fmt"
	"os"
	"syscall"
)

func mapFile(f *os.File) (MMap, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, fmt.Errorf("mmap: cannot map an empty file %q", f.Name())
	}

	fd := int(f.Fd())   // #nosec G115 -- a file descriptor is a small non-negative int
	length := int(size) // #nosec G115 -- the file size fits an int on the 64-bit targets we build
	b, err := syscall.Mmap(fd, 0, length, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap: %w", err)
	}
	return MMap(b), nil
}

func unmap(m MMap) error {
	if len(m) == 0 {
		return nil
	}
	return syscall.Munmap(m)
}
