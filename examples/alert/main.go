// Command alert shows two input alerts and prints what came back: the first
// is meant to be cancelled (Escape), the second answered (type, Return). CI
// drives it under xvfb with xdotool.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/crgimenes/native/alert"
)

func init() { runtime.LockOSThread() }

func main() {
	for i := range 2 {
		res, err := alert.Show(alert.Options{
			Title:   "native/alert",
			Message: "Type something",
			Input:   true,
			Buttons: []alert.Button{{Title: "OK"}, {Title: "Cancel", Cancel: true}},
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "alert:", err)
			os.Exit(1)
		}
		if res.Button != 0 {
			fmt.Printf("alert %d: canceled\n", i+1)
			continue
		}
		fmt.Printf("alert %d: %s\n", i+1, res.Text)
	}
}
