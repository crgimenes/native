// Command power demonstrates github.com/crgimenes/native/power: it asks the OS
// to keep the system awake, holds the inhibition for a few seconds, then releases
// it. While it is held you can confirm it is live with `pmset -g assertions` on
// macOS or `powercfg /requests` on Windows. Run it with:
//
//	go run ./examples/power
package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/crgimenes/native/power"
)

func main() {
	tok, err := power.PreventSleep("native power example")
	if errors.Is(err, power.ErrUnsupported) {
		log.Fatalf("power is not supported on this platform: %v", err)
	}
	if err != nil {
		log.Fatalf("prevent sleep: %v", err)
	}

	const hold = 5 * time.Second
	fmt.Printf("system sleep inhibited for %s — check with `pmset -g assertions` (macOS) or `powercfg /requests` (Windows)\n", hold)
	time.Sleep(hold)

	err = tok.Release()
	if err != nil {
		log.Fatalf("release: %v", err)
	}
	fmt.Println("inhibition released; the system may idle-sleep again")
}
