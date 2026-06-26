# mmap

Memory-map a file, cgo-free. The mapped region is a `[]byte` you read and write
directly; the OS writes changes back to the file. On Unix this is a thin wrapper
over `syscall.Mmap` (no purego); on Windows it binds the file-mapping API through
purego.

```go
import "github.com/crgimenes/native/mmap"

f, _ := os.OpenFile("data.bin", os.O_CREATE|os.O_RDWR, 0o644)
f.Truncate(4096)          // mmap can't map an empty file
m, _ := mmap.Map(f)
copy(m, []byte("hello"))  // writes the file
m.Unmap()
```

## API

| Func | Description |
| --- | --- |
| `Map(f *os.File) (MMap, error)` | Map all of `f` read-write (shared). `f` must be opened read-write and sized; an empty file fails. |
| `(MMap) Unmap() error` | Release the mapping. Don't use the slice afterward. |
| `MMap` | `[]byte` aliasing the file's bytes. Don't append or reslice past its length. |
| `ErrUnsupported` | Sentinel returned by a platform with no backend. |

The byte slice is the whole API surface. No native handles cross the boundary.

## Platforms

| OS | Backend | Status |
| --- | --- | --- |
| macOS | `syscall.Mmap` (stdlib) | ✅ |
| Linux | `syscall.Mmap` (stdlib) | ✅ |
| Windows | `CreateFileMappingW` + `MapViewOfFile` (purego) | ✅ builds + CI |

Check the unsupported case with `errors.Is(err, mmap.ErrUnsupported)`.

## Sharing between processes

Map the same file from two processes and they share the same physical pages, so a
write by one is visible to the other immediately — a fast IPC channel. No `Flush`
is needed: that only forces dirty pages to *disk*, which sharing doesn't care
about. On Linux put the file in `/dev/shm` (tmpfs) for a RAM-only backing with no
disk I/O.

`mmap` gives you the shared bytes, not synchronization. Coordinate access
yourself — `sync/atomic` on the mapped bytes works across processes (it's the same
memory), e.g. a lock-free ring buffer with atomic indices. `TestSharedAcrossMappings`
checks the cross-mapping coherence this relies on.

## Scope

Maps the **whole** file, **read-write**, **shared**. Partial maps (offset/length),
read-only maps, and an explicit `Flush` (msync / `FlushViewOfFile`) are not here
yet — the OS writes dirty pages back on `Unmap` and during normal writeback, so a
basic round trip needs none of them. They can be added behind an `Options` /
`Flush` when a real case needs them.

## Example

A runnable demo (map, write, read back) lives in
[`examples/mmap`](../examples/mmap):

```bash
go run ./examples/mmap
```

## Conventions

Part of [native](../README.md); follows the shared shape — public API in a
tag-free `mmap.go`, a shared `_unix.go` backend for macOS/Linux, `_windows.go`,
and `_other.go` so every `GOOS` builds.
