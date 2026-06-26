// Command mmap demonstrates github.com/crgimenes/native/mmap: it maps a file,
// edits it through the byte slice, unmaps, and reads the change back from disk.
// Run with:
//
//	go run ./examples/mmap
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/crgimenes/native/mmap"
)

func main() {
	f, err := os.CreateTemp("", "native-mmap-*.bin")
	if err != nil {
		log.Fatalf("create temp file: %v", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer func() { _ = f.Close() }()

	// mmap can't map an empty file, so give it a size first.
	err = f.Truncate(64)
	if err != nil {
		log.Fatalf("truncate: %v", err)
	}

	m, err := mmap.Map(f)
	if errors.Is(err, mmap.ErrUnsupported) {
		log.Fatalf("mmap is not supported on this platform: %v", err)
	}
	if err != nil {
		log.Fatalf("map: %v", err)
	}

	const msg = "written through the mmap"
	copy(m, []byte(msg))

	err = m.Unmap()
	if err != nil {
		log.Fatalf("unmap: %v", err)
	}

	// Read it back from the file to prove the write reached the file.
	got := make([]byte, len(msg))
	_, err = f.ReadAt(got, 0)
	if err != nil {
		log.Fatalf("read back: %v", err)
	}
	fmt.Printf("file now starts with: %q\n", got)
}
