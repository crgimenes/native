// Command bookmark demonstrates github.com/crgimenes/native/bookmark: it
// creates a persistence token for a file, resolves the token back into a path
// (starting the access grant), and releases the grant. Run it with:
//
//	go run ./examples/bookmark
//
// Outside the macOS App Sandbox (and on Windows/Linux) the token is just the
// path; inside the sandbox it is real security-scoped bookmark data. Either
// way the calling code is identical — store the token, Resolve on the next
// launch, release when done. No UI and no main-thread requirement.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/crgimenes/native/bookmark"
)

func main() {
	dir, err := os.MkdirTemp("", "bookmark-example-")
	if err != nil {
		log.Fatalf("tempdir: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	file := filepath.Join(dir, "note.txt")
	err = os.WriteFile(file, []byte("remember me\n"), 0o600)
	if err != nil {
		log.Fatalf("write: %v", err)
	}

	// In a real app this happens right after the user picks the file
	// (e.g. with filedialog.Open); the token then goes into the app's
	// persistent state.
	token, err := bookmark.Create(file)
	if err != nil {
		log.Fatalf("create: %v", err)
	}
	fmt.Printf("token: %d bytes (kind %q)\n", len(token), token[0])

	// On the next launch: turn the stored token back into a usable path.
	path, release, err := bookmark.Resolve(token)
	if err != nil {
		log.Fatalf("resolve: %v", err)
	}
	defer release()

	// #nosec G304 -- path comes from a security-scoped bookmark the user
	// granted; resolving one and reading through it is what this example shows.
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read via resolved path: %v", err)
	}
	fmt.Printf("resolved: %s\ncontent: %s", path, data)
}
