// Command tray demonstrates github.com/crgimenes/native/tray: it puts an icon
// with a small menu in the system tray / menu bar and logs when items are
// chosen. Run it with:
//
//	go run ./examples/tray
//
// It builds a PNG icon in memory (no asset file) and passes it as Config.Icon.
// macOS renders that PNG in the menu bar; Windows currently shows the default
// application icon instead (honoring a custom PNG there is a follow-up), so the
// icon "shows when possible" without any per-platform code in the caller.
//
// The tray owns the UI event loop, so main locks the OS thread and Run blocks
// until "Quit" (or Stop) ends it. Set TRAY_AUTOCLOSE=1 to have it stop itself
// after a couple of seconds — handy for a non-interactive smoke test.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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
		Icon:    discPNG(),
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

// discPNG returns a small filled-circle PNG so the example has an icon without
// shipping an asset file.
func discPNG() []byte {
	const s = 44
	img := image.NewRGBA(image.Rect(0, 0, s, s))
	cx, cy, r := float64(s)/2, float64(s)/2, float64(s)/2-1
	for y := range s {
		for x := range s {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, color.RGBA{R: 45, G: 124, B: 240, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		log.Fatalf("encode icon: %v", err)
	}
	return buf.Bytes()
}
