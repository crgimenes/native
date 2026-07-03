// Command clipboard demonstrates github.com/crgimenes/native/clipboard:
// it writes text to the system clipboard, reads it back, and restores whatever
// was there before. Run it with:
//
//	go run ./examples/clipboard
package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/crgimenes/native/clipboard"
)

func main() {
	// Save the current content so we can put it back afterwards.
	previous, err := clipboard.ReadText()
	if errors.Is(err, clipboard.ErrUnsupported) {
		log.Fatalf("clipboard is not supported on this platform yet: %v", err)
	}
	if err != nil {
		log.Fatalf("read clipboard: %v", err)
	}

	const msg = "hello from native/clipboard"
	err = clipboard.WriteText(msg)
	if err != nil {
		log.Fatalf("write clipboard: %v", err)
	}

	got, err := clipboard.ReadText()
	if err != nil {
		log.Fatalf("read clipboard back: %v", err)
	}
	fmt.Printf("wrote %q\nread  %q\nmatch %v\n", msg, got, got == msg)

	// Be a good citizen: restore the user's previous clipboard.
	err = clipboard.WriteText(previous)
	if err != nil {
		log.Fatalf("restore clipboard: %v", err)
	}
	fmt.Println("restored the previous clipboard content")
}
