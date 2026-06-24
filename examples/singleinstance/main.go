// Command singleinstance demonstrates github.com/crgimenes/native/singleinstance.
// Run it once to become the primary; run it again (ideally with arguments) in
// another terminal and the second launch forwards its arguments to the first and
// exits. Press Ctrl-C to quit the primary.
//
//	go run ./examples/singleinstance
//	go run ./examples/singleinstance hello world   # in a second terminal
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/crgimenes/native/singleinstance"
)

const appID = "com.example.native-singleinstance-demo"

func main() {
	inst, err := singleinstance.Acquire(appID, singleinstance.Options{
		OnMessage: func(args []string) {
			fmt.Printf("another launch started with args: %v\n", args)
		},
	})
	switch {
	case errors.Is(err, singleinstance.ErrAlreadyRunning):
		// We are a secondary launch: hand our args to the primary and exit.
		if e := singleinstance.Send(appID, os.Args[1:]); e != nil {
			log.Fatalf("forward args to running instance: %v", e)
		}
		fmt.Println("an instance is already running; forwarded our args to it")
		return
	case errors.Is(err, singleinstance.ErrUnsupported):
		log.Fatalf("singleinstance is not supported on this platform: %v", err)
	case err != nil:
		log.Fatal(err)
	}
	defer func() { _ = inst.Release() }()

	fmt.Println("primary instance running. Launch me again (with args) in another")
	fmt.Println("terminal to see the hand-off. Press Ctrl-C to quit.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
	fmt.Println("\nshutting down")
}
