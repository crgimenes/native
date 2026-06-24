// Command openurl demonstrates github.com/crgimenes/native/openurl: it opens a
// web page in the default browser and reveals a file in the file manager. Run it
// with:
//
//	go run ./examples/openurl
//
// It actually launches the browser and file manager, so run it interactively.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/crgimenes/native/openurl"
)

func main() {
	const page = "https://example.com"
	err := openurl.Open(page)
	if err != nil {
		if errors.Is(err, openurl.ErrUnsupported) {
			log.Fatalf("openurl is not supported on this platform yet: %v", err)
		}
		log.Fatalf("open %q: %v", page, err)
	}
	fmt.Printf("opened %s in the default browser\n", page)

	// Make a throwaway file and reveal it in the file manager.
	f, err := os.CreateTemp("", "native-openurl-*.txt")
	if err != nil {
		log.Fatalf("create temp file: %v", err)
	}
	_, _ = f.WriteString("revealed by native/openurl\n")
	_ = f.Close()
	defer func() { _ = os.Remove(f.Name()) }()

	err = openurl.Reveal(f.Name())
	if err != nil {
		log.Fatalf("reveal %q: %v", f.Name(), err)
	}
	fmt.Printf("revealed %s in the file manager\n", filepath.Clean(f.Name()))
}
