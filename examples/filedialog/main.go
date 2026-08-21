// Command filedialog demonstrates github.com/crgimenes/native/filedialog: it
// shows the native open panel, the native save panel, then the native
// choose-directory panel. Run it with:
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
	saveMsg := "save: canceled (or platform not supported yet)"
	if saved != "" {
		saveMsg = fmt.Sprintf("save: %s (nothing is written; the panel only picks the path)", saved)
	}
	fmt.Println(saveMsg)

	dir := filedialog.PickDirectory(filedialog.Options{
		Title: "Pick any directory",
	})
	dirMsg := "directory: canceled (or platform not supported yet)"
	if dir != "" {
		dirMsg = "directory: " + dir
	}
	fmt.Println(dirMsg)
}
