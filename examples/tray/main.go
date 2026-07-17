// Command tray demonstrates github.com/crgimenes/native/tray: it puts an icon
// with a small menu in the system tray / menu bar and logs when items are
// chosen. Run it with:
//
//	go run ./examples/tray
//
// The tray owns the UI event loop, so main locks the OS thread and Run blocks
// until "Quit" (or Stop) ends it. Set TRAY_AUTOCLOSE=1 to have it stop itself
// after a couple of seconds — handy for a non-interactive smoke test.
package main

import (
	"log"
	"os"
	"runtime"
	"time"

	"github.com/crgimenes/native/tray"
)

func main() {
	runtime.LockOSThread()

	if os.Getenv("TRAY_AUTOCLOSE") != "" {
		go func() {
			time.Sleep(2 * time.Second)
			log.Println("auto-close: stopping the tray")
			tray.Stop()
		}()
	}

	cfg := tray.Config{
		Title:   "native",
		Tooltip: "native tray example",
		Items: []tray.Item{
			{Title: "Say hello", OnClick: func() { log.Println("hello from the tray") }},
			{Separator: true},
			{Title: "Disabled item", Disabled: true},
			{Title: "Quit", OnClick: func() { log.Println("quit"); tray.Stop() }},
		},
	}

	log.Println("starting tray — look in the menu bar (macOS) or notification area (Windows)")
	err := tray.Run(cfg)
	if err != nil {
		log.Fatalf("tray: %v", err)
	}
	log.Println("tray stopped")
}
