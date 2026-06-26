package mmap_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/crgimenes/native/mmap"
)

// TestRoundTrip maps a sized file, writes through the mapping, unmaps, and reads
// the file back from disk to confirm the write landed -- exercising the real mmap
// path (syscall.Mmap on Unix, the file-mapping API on Windows).
func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.bin")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	const size = 4096
	err = f.Truncate(size)
	if err != nil {
		t.Fatal(err)
	}

	m, err := mmap.Map(f)
	if errors.Is(err, mmap.ErrUnsupported) {
		t.Skipf("mmap unsupported here: %v", err)
	}
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if len(m) != size {
		t.Fatalf("len(m) = %d, want %d", len(m), size)
	}

	want := []byte("hello mmap — café ✓")
	copy(m, want)

	err = m.Unmap()
	if err != nil {
		t.Fatalf("Unmap: %v", err)
	}

	got := make([]byte, len(want))
	_, err = f.ReadAt(got, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("read back %q, want %q", got, want)
	}
}

// TestMapEmptyFails confirms mapping an empty file returns an error instead of
// panicking.
func TestMapEmptyFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.bin")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	_, err = mmap.Map(f)
	if errors.Is(err, mmap.ErrUnsupported) {
		t.Skipf("mmap unsupported here: %v", err)
	}
	if err == nil {
		t.Fatal("Map of an empty file should fail")
	}
}

// TestSharedAcrossMappings proves the property the package exists for: two
// independent mappings of the SAME file -- two os.File handles, as two processes
// would each have -- share memory, so a write through one mapping is visible
// through the other in both directions. MAP_SHARED on Unix and a file-backed
// mapping on Windows both guarantee this, with no flush involved. This is the
// fast inter-process channel.
func TestSharedAcrossMappings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shm.bin")

	f1, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f1.Close() }()
	const size = 256
	err = f1.Truncate(size)
	if err != nil {
		t.Fatal(err)
	}

	m1, err := mmap.Map(f1)
	if errors.Is(err, mmap.ErrUnsupported) {
		t.Skipf("mmap unsupported here: %v", err)
	}
	if err != nil {
		t.Fatalf("Map f1: %v", err)
	}
	defer func() { _ = m1.Unmap() }()

	// A second, independent handle and mapping of the same file, standing in for
	// a second process.
	f2, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f2.Close() }()
	m2, err := mmap.Map(f2)
	if err != nil {
		t.Fatalf("Map f2: %v", err)
	}
	defer func() { _ = m2.Unmap() }()

	// Write through the first mapping; the second must see it.
	want := []byte("shared via mmap — café ✓")
	copy(m1, want)
	got := m2[:len(want)]
	if string(got) != string(want) {
		t.Fatalf("second mapping sees %q, want %q", got, want)
	}

	// And the reverse direction, at an offset.
	m2[100] = 0x42
	if m1[100] != 0x42 {
		t.Fatalf("first mapping sees m1[100] = %#x, want 0x42", m1[100])
	}
}
