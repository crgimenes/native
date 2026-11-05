// Command notify posts one desktop notification:
//
//	go run ./examples/notify [title [body]]
package main

import (
	"fmt"
	"os"

	"github.com/crgimenes/native/notify"
)

func main() {
	m := notify.Message{Title: "native/notify", Body: "Hello from a cgo-free Go binary."}
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Println("usage: notify [title [body]]\n\nPosts one desktop notification.\n\nexample: notify 'Build done' 'all green'")
		return
	}
	if len(os.Args) > 1 {
		m.Title = os.Args[1]
	}
	if len(os.Args) > 2 {
		m.Body = os.Args[2]
	}
	err := notify.Send(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("notification sent")
}
