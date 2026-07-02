// Command filedialog demonstrates github.com/crgimenes/native/filedialog: it
// shows the native open panel, then the native save panel. Run it with:
//
//	go run ./examples/filedialog
//
// It opens modal panels, so run it interactively. The panels must run on the
// main thread; init pins the main goroutine there before main starts.
package main

import (
	"fmt"
	"runtime"

	"github.com/crgimenes/native/filedialog"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	opened := filedialog.Open(filedialog.Options{
		Title: "Pick any file",
	})
	openMsg := "open: canceled (or platform not supported yet)"
	if opened != "" {
		openMsg = "open: " + opened
	}
	fmt.Println(openMsg)

	saved := filedialog.Save(filedialog.Options{
		Title:    "Save example note",
		Filename: "note.txt",
	})
	if saved == "" {
		fmt.Println("save: canceled (or platform not supported yet)")
		return
	}
	fmt.Printf("save: %s (nothing is written; the panel only picks the path)\n", saved)
}
