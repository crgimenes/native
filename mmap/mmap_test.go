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
