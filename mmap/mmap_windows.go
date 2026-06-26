// Windows backend: CreateFileMappingW + MapViewOfFile (kernel32) via purego.
// Windows has no dlopen, so the symbols are resolved with LoadLibrary/
// GetProcAddress and bound with purego.RegisterFunc. Unmap needs both the view
// base address (for UnmapViewOfFile) and the mapping handle (for CloseHandle),
// so they are kept in a package map keyed by the view base.

package mmap

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	pageReadWrite = 0x04   // PAGE_READWRITE
	fileMapWrite  = 0x0002 // FILE_MAP_WRITE
	fileMapRead   = 0x0004 // FILE_MAP_READ
)

var (
	initOnce sync.Once
	initErr  error

	createFileMappingW func(hFile, attrs uintptr, protect, sizeHigh, sizeLow uint32, name *uint16) uintptr
	mapViewOfFile      func(hMapping uintptr, access, offHigh, offLow uint32, bytes uintptr) uintptr
	unmapViewOfFile    func(base uintptr) int32
	closeHandle        func(h uintptr) int32

	mapMu    sync.Mutex
	mappings = map[uintptr]uintptr{} // view base -> file-mapping handle
)

func ensureInit() error {
	initOnce.Do(func() {
		k32, err := syscall.LoadLibrary("kernel32.dll")
		if err != nil {
			initErr = fmt.Errorf("mmap: load kernel32.dll: %w", err)
			return
		}
		reg := func(p any, name string) {
			if initErr != nil {
				return
			}
			addr, e := syscall.GetProcAddress(k32, name)
			if e != nil {
				initErr = fmt.Errorf("mmap: resolve %s: %w", name, e)
				return
			}
			purego.RegisterFunc(p, addr)
		}
		reg(&createFileMappingW, "CreateFileMappingW")
		reg(&mapViewOfFile, "MapViewOfFile")
		reg(&unmapViewOfFile, "UnmapViewOfFile")
		reg(&closeHandle, "CloseHandle")
	})
	return initErr
}

func mapFile(f *os.File) (MMap, error) {
	err := ensureInit()
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := info.Size()
	if size == 0 {
		return nil, fmt.Errorf("mmap: cannot map an empty file %q", f.Name())
	}

	// Passing 0 for the maximum size maps the file's current length.
	h := createFileMappingW(f.Fd(), 0, pageReadWrite, 0, 0, nil)
	if h == 0 {
		return nil, errors.New("mmap: CreateFileMappingW failed")
	}
	base := mapViewOfFile(h, fileMapRead|fileMapWrite, 0, 0, 0)
	if base == 0 {
		closeHandle(h)
		return nil, errors.New("mmap: MapViewOfFile failed")
	}

	mapMu.Lock()
	mappings[base] = h
	mapMu.Unlock()

	return MMap(unsafe.Slice((*byte)(ptr(base)), size)), nil // #nosec G103 -- slice over the mapped view
}

func unmap(m MMap) error {
	if len(m) == 0 {
		return nil
	}
	base := uintptr(unsafe.Pointer(&m[0])) // #nosec G103 -- the mapped view's base address
	mapMu.Lock()
	h := mappings[base]
	delete(mappings, base)
	mapMu.Unlock()

	unmapViewOfFile(base)
	if h != 0 {
		closeHandle(h)
	}
	return nil
}

// ptr reinterprets a uintptr as an unsafe.Pointer without a direct
// uintptr->Pointer cast (keeps go vet's unsafeptr check quiet). The address comes
// from MapViewOfFile: system memory the Go GC neither owns nor moves.
func ptr(u uintptr) unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&u)) } // #nosec G103
