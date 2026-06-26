// Package mmap memory-maps files, cgo-free. The mapped region is a []byte you
// read and write directly; the OS writes changes back to the file.
//
//	f, _ := os.OpenFile("data.bin", os.O_CREATE|os.O_RDWR, 0o644)
//	f.Truncate(4096)        // mmap can't map an empty file
//	m, _ := mmap.Map(f)
//	copy(m, []byte("hello"))
//	m.Unmap()
//
// On Unix this is a thin wrapper over syscall.Mmap (no purego needed); on Windows
// it binds the file-mapping API through purego.
package mmap

import (
	"errors"
	"os"
)

// ErrUnsupported is returned by Map on a platform with no backend.
var ErrUnsupported = errors.New("mmap: not supported on this platform")

// MMap is a memory-mapped region. It aliases the file's bytes: writing to the
// slice writes the file. Don't append to it or reslice past its length, and
// don't use it after Unmap.
type MMap []byte

// Map maps the whole of f into memory for reading and writing (shared, so writes
// reach the file). f must be opened read-write and already sized, e.g. with
// Truncate; mapping an empty file fails. Unmap the result when done.
//
// yagni: whole-file, read-write, shared. Add an Options{Offset, Length, ReadOnly}
// only when a real case needs a partial or read-only mapping.
func Map(f *os.File) (MMap, error) { return mapFile(f) }

// Unmap releases the mapping. The MMap must not be used afterward.
func (m MMap) Unmap() error { return unmap(m) }
